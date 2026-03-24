# GSCALE-ZEBRA

A multi-module (Go workspace) system built for Scale + Zebra RFID + Telegram Bot + ERPNext integration.

## Abstract
This project integrates real-time weighing, RFID tagging, and ERP documentation into a single operational pipeline. The core idea is to move from physical measurement (quantity) to ERPNext `Material Receipt` draft creation, then print the exact same EPC via Zebra, and finally submit the document through a state-driven reliable workflow rather than fragmented manual steps.

As an applied institute project, this system demonstrates:
- real-time signal processing;
- stability-based triggering logic;
- atomic shared-state updates in a multi-process setup;
- external service integration (Telegram, ERP API, Zebra USB);
- Linux service operation using systemd.

## Contents
- [1. Problem Statement and Goal](#1-problem-statement-and-goal)
- [2. System Architecture](#2-system-architecture)
- [3. Module Overview](#3-module-overview)
- [4. Data Model (bridge state)](#4-data-model-bridge-state)
- [5. Core Algorithms](#5-core-algorithms)
- [6. Runtime Workflow](#6-runtime-workflow)
- [7. Installation and Run](#7-installation-and-run)
- [8. Configuration](#8-configuration)
- [9. Commands and Control](#9-commands-and-control)
- [10. Logging, Monitoring, and Failures](#10-logging-monitoring-and-failures)
- [11. Testing and Verification](#11-testing-and-verification)
- [12. Limitations and Future Work](#12-limitations-and-future-work)

## 1. Problem Statement and Goal
### Problem
Operationally, the following steps are often disconnected:
1. Read product weight from a scale.
2. Write and verify EPC on an RFID tag.
3. Create an operational ERP document (draft).

When these are performed separately, human error, latency, and traceability issues increase.

### Goal
Build one integrated system that:
- detects a new cycle when scale reading becomes stable;
- tracks Zebra result and status;
- controls ERP draft creation based on batch state;
- keeps all component states in one shared state file.

## 2. System Architecture
The repository is organized with `go.work` and 5 main modules:

- `scale`: real-time worker + TUI, scale and Zebra orchestration.
- `bot`: Telegram bot, ERP integration, batch session control.
- `bridge`: atomic JSON shared-state store.
- `core`: stable-quantity to EPC trigger logic.
- `zebra`: CLI diagnostics and service commands.

### High-level flow
```text
[Scale serial/bridge] -> [scale worker] -> [bridge_state.json] <- [bot worker]
                                 |                      |
                                 v                      v
                           [Zebra USB I/O]         [ERPNext API]
                                 |
                                 v
                            [EPC / VERIFY]
```

### Design choices
- Module separation improves maintainability and testing.
- Shared-state file enables loose coupling between services.
- File-lock + atomic rename reduces race and partial-write risks.

## 3. Module Overview
### `scale` module
Main responsibilities:
- auto-detect serial port (`/dev/serial/by-id/*`, `ttyUSB*`, `ttyACM*`);
- parse scale frames (`kg/g/lb/oz`, negative formats, stable/unstable markers);
- fallback to HTTP bridge when serial is unavailable;
- poll Zebra status;
- provide operator TUI (`q`, `e`, `r`);
- write `scale` + `zebra` snapshots to bridge state;
- consume `print_request` commands from bridge state and execute printer actions.

### `bot` module
Main responsibilities:
- process Telegram commands: `/start`, `/batch`, `/log`, `/epc`;
- item/warehouse selection from ERP using inline queries;
- open/stop batch sessions (`Material Receipt` callback);
- wait for stable quantity from bridge state;
- create `Stock Entry` (`Material Receipt`) draft in ERPNext;
- write `print_request` commands into bridge state;
- submit or delete the draft based on print result;
- keep in-memory EPC history for current bot session and export it as `.txt` (`/epc`).

### `bridge` module
Main responsibilities:
- store `bridge_state.json` as single source of truth;
- use lock file for exclusive updates;
- write temp file then rename for atomic state replacement.

### `core` module
Main responsibilities:
- provide shared EPC generation logic;
- generate unique 24-hex EPC for each new cycle.

### `zebra` module
Main responsibilities:
- discover printer and query status/SGD settings;
- run RFID encode/read tests;
- calibration and self-check operations.

## 4. Data Model (bridge state)
Default file:
- `/tmp/gscale-zebra/bridge_state.json`

The snapshot has 4 main sections:
- `scale`: source, port, weight, unit, stable, error, updated_at
- `zebra`: connected, device state, media state, last_epc, verify, action, error, updated_at
- `batch`: active, chat_id, item_code, item_name, warehouse, updated_at
- `print_request`: epc, qty, unit, item_code, item_name, status, error, requested_at, updated_at

Example:
```json
{
  "scale": {
    "source": "serial",
    "port": "/dev/ttyUSB0",
    "weight": 1.25,
    "unit": "kg",
    "stable": true,
    "updated_at": "2026-02-20T10:10:10.123Z"
  },
  "zebra": {
    "connected": true,
    "device_path": "/dev/usb/lp0",
    "last_epc": "3034257BF7194E406994036B",
    "verify": "MATCH",
    "action": "encode",
    "updated_at": "2026-02-20T10:10:10.456Z"
  },
  "batch": {
    "active": true,
    "chat_id": 123456789,
    "item_code": "ITEM-001",
    "item_name": "Green Tea",
    "warehouse": "Stores - A",
    "updated_at": "2026-02-20T10:10:09.999Z"
  },
  "updated_at": "2026-02-20T10:10:10.500Z"
}
```

## 5. Core Algorithms
### 5.1 Stable quantity cycle detection
Parameters:
- `StableFor = 1s`
- `Epsilon = 0.005`
- `MinWeight = 0.0`

Rules:
1. If `weight <= 0` or invalid, state may reset, but `0` is not a required condition for a new cycle.
2. A stable point is accepted when the stream stays within `|w - candidate| <= Epsilon` for `StableFor`.
3. A new cycle opens only after a real `movement/unstable` phase is observed after the previous stable point.
4. The next stable point may be greater, smaller, or nearly equal to the previous one.
5. The practical model is `stable -> movement -> next stable`.

### 5.2 EPC generation
24-character uppercase hex format:
- prefix: `30`
- time-based part: derived from 56-bit slice of Unix nanoseconds
- tail: mixed from time atom + sequence + process salt

Outcome:
- very low collision risk even with frequent triggers;
- restart-safe differentiation due to per-process salt.

### 5.3 Zebra encode and verify
Encode flow inside `scale`:
1. Apply RFID ultra settings (`rfid.enable`, power, tries, etc.).
2. Write EPC via ZPL stream (`^RFW,H,,,A`).
3. Sample `rfid.error.response` and infer `WRITTEN/NO TAG/ERROR/UNKNOWN`.
4. Optionally run additional readback validation.
5. Write `verify` and `last_epc` to bridge state.

Successful `verify` values:
- `MATCH`, `OK`, `WRITTEN`

### 5.4 ERP draft creation criteria in bot
Bot batch loop:
1. Wait for stable positive quantity from bridge state.
2. Generate EPC for the current cycle.
3. Create ERP `Material Receipt` draft using that EPC.
4. Write a `print_request` for the worker with the same EPC.
5. Submit the draft on print success.
6. Delete the draft on print failure.

Note:
- if ERP reports a duplicate barcode before final print, a fresh candidate EPC is generated and retried.

## 6. Runtime Workflow
### 6.1 End-to-end batch sequence
```text
Operator -> Telegram: /batch
Bot -> ERP: item/warehouse search
Operator -> Bot: Material Receipt
Bot -> bridge_state: batch.active=true
Scale -> bridge_state: live qty/stable
Bot <- bridge_state: stable qty
Bot -> ERP: Stock Entry (Material Receipt) draft
Bot -> bridge_state: print_request pending
Scale <- bridge_state: print_request
Scale -> Zebra: EPC encode/print
Scale -> bridge_state: print_request done/error + zebra status
Bot <- bridge_state: print result
Bot -> ERP: submit (success) / delete (error)
Bot -> Telegram: status update
```

### 6.2 Additional service commands
- `/log`: sends `logs/bot` and `logs/scale` files to Telegram as documents.
- `/epc`: sends all EPC values used for successful drafts in current bot session as `.txt` document.

## 7. Installation and Run
### 7.1 Requirements
- Linux (tested on Ubuntu/Arch style hosts)
- Go `1.25`
- USB serial scale device
- Zebra USB LP printer (`/dev/usb/lp*`)
- Telegram bot token
- ERPNext API key/secret

### 7.2 Development mode
From repo root:
```bash
make build
make test
```

Scale + auto bot:
```bash
make run SCALE_DEVICE=/dev/ttyUSB0 ZEBRA_DEVICE=/dev/usb/lp0
```

Scale only:
```bash
make run-scale SCALE_DEVICE=/dev/ttyUSB0 ZEBRA_DEVICE=/dev/usb/lp0
```

Bot only:
```bash
cd bot
cp .env.example .env
# fill token and ERP credentials
go run ./cmd/bot
```

### 7.3 Systemd autostart (repo mode)
```bash
make autostart-install
make autostart-status
```

### 7.4 Release package
```bash
make release
# or
make release-all
```

Artifacts are generated in `dist/` as Linux tarballs.

## 8. Configuration
### 8.1 Bot (`bot/.env`)
Required:
- `TELEGRAM_BOT_TOKEN`
- `ERP_URL`
- `ERP_API_KEY`
- `ERP_API_SECRET`

Optional:
- `BRIDGE_STATE_FILE` (default: `/tmp/gscale-zebra/bridge_state.json`)

### 8.2 Scale (flags)
Main flags:
- `--device`, `--baud`, `--baud-list`
- `--bridge-url`, `--bridge-interval`, `--no-bridge`
- `--zebra-device`, `--zebra-interval`, `--no-zebra`
- `--bot-dir`, `--no-bot`
- `--bridge-state-file`

### 8.3 Deploy env (systemd)
`deploy/config/scale.env.example`:
- `SCALE_DEVICE`
- `ZEBRA_DEVICE`
- `BRIDGE_STATE_FILE`

`deploy/config/bot.env.example`:
- `TELEGRAM_BOT_TOKEN`
- `ERP_URL`
- `ERP_API_KEY`
- `ERP_API_SECRET`
- `BRIDGE_STATE_FILE`

## 9. Commands and Control
### 9.1 Make targets
- `make run`: scale TUI (with bot auto-start)
- `make run-scale`: scale only
- `make run-bot`: bot only
- `make test`: run tests in all modules
- `make autostart-install|status|restart|stop`

### 9.2 Bot commands
- `/start`: ERP connectivity check
- `/batch`: batch selection and start flow
- `/log`: send workflow logs
- `/epc`: send session EPC list as `.txt`

### 9.3 Scale TUI keys
- `q`: quit
- `e`: manual encode+print
- `r`: manual RFID read

### 9.4 Zebra utility
```bash
cd zebra
go run . help
```
Main commands:
- `list`, `status`, `settings`, `setvar`, `raw-getvar`
- `print-test`, `epc-test`, `read-epc`, `calibrate`, `self-check`

## 10. Logging, Monitoring, and Failures
Log folders:
- `logs/scale/`
- `logs/bot/`

Important behavior:
- each process startup clears its own log folder and starts a fresh session.

### Typical failures
1. `device busy`:
- usually another process is already using printer port.

2. `serial device not found`:
- provide explicit `SCALE_DEVICE` or verify device mapping/udev.

3. `ERP auth/http error`:
- verify URL, API key, and API secret in `.env`.

4. `epc timeout`:
- may be caused by Zebra timing delay or state update timing.

## 11. Testing and Verification
Unit tests are available for:
- `core`: stable detector and EPC uniqueness
- `bridge`: atomic store update/read
- `scale`: parser, frame extraction, zebra stream building
- `bot`: command parsing, ERP payload, log discovery, EPC history

Current status: all tests pass
```bash
make test
```

## 12. Limitations and Future Work
### Current limitations
- primary target platform is Linux;
- `Receipt` callback flow is currently a placeholder;
- EPC history (`/epc`) is in process memory only (cleared on restart);
- draft creation currently proceeds when EPC exists even if verify is not successful.

### Suggested next steps
1. Add strict/lenient verify policy mode.
2. Move `/epc` history to persistent storage (SQLite or append-only log).
3. Add bridge-state schema versioning and migration path.
4. Add metrics and health endpoints.
5. Add end-to-end integration tests (mock ERP + mock bridge + trace replay).

---

If needed, this can be extended into a full academic documentation package under `docs/`:
- `docs/architecture.md`
- `docs/algorithm.md`
- `docs/experiment-results.md`
- `docs/appendix-api-contracts.md`
