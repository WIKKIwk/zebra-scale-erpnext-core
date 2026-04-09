# Mobile App Context

## Current Goal

- `mobile_app` Flutter klienti `gscale-zebra` backendiga ulanadi.
- Maqsad: Wi-Fi/tarmoq orqali ishlayotgan serverdan holat o'qish.
- Hozirgi fokus: `mobileapi` va `polygon` bilan Android emulator ichida ulanishni ishlatish.

## Important Rules From User

- Ortiqcha yo'l tanlanmaydi.
- Login flow hozir kerak emas.
- Bitta buyruq bilan dev oqimi ishga tushishi kerak.
- Asosiy test oqimi Android emulator ichida bo'ladi.

## Current Backend State

- `internal/mobileapi/server.go`
  - `GET /healthz` ishlaydi.
  - `GET /v1/mobile/monitor/state` endi login talab qilmaydi.
  - `GET /v1/mobile/profile` endi login talab qilmaydi.
- `polygon` fake scale/printer holatini beradi.

## Current Mobile App State

- `mobile_app/lib/main.dart`
  - `Connect` bosilganda:
    - `GET /healthz`
    - `GET /v1/mobile/monitor/state`
  - Olingan javob UI kartalariga bosiladi.
- `device_preview`
  - desktop/web uchun yoqilgan
  - android/ios uchun o'chirilgan

## Make Targets

- Root repo:
  - `make run-dev`
    - `mobileapi`
    - `polygon`
    - `mobile_app` android run
- `mobile_app`:
  - `make run-android`
  - `make run`
  - `make analyze`
  - `make test`

## Android Emulator Notes

- Emulator IDs:
  - `gscale_api35`
  - `gscale_atd35`
- Default android emulator:
  - `gscale_atd35`
- `mobile_app/Makefile`
  - android run `--profile` rejimda yuradi
  - `adb reverse` bilan `8081` va `18000` portlar ulanadi

## Networking Notes

- Emulator ichida Wi-Fi yoqilgan.
- Emulator `AndroidWifi` AP ni ko'radi va ulanadi.
- Emulator hostga chiqishi tasdiqlangan:
  - `10.0.2.2`
- Hozirgi ulanish modeli:
  - emulator -> `mobileapi`
  - emulator -> `polygon`

## Verified Items

- `flutter analyze` o'tgan.
- `flutter test` o'tgan.
- `go test ./internal/mobileapi ./cmd/mobileapi` o'tgan.
- `GET /v1/mobile/monitor/state` 200 qaytargan.

## Known Problems

- Android emulator hali ham ba'zan beqaror bo'lishi mumkin.
- Avvalgi og'ir `google_apis` image o'rniga yengilroq `ATD` ishlatilmoqda.
- Agar emulator yana osilsa, keyingi fokus emulator stability bo'ladi, login emas.

## Files Touched Recently

- `Makefile`
- `internal/mobileapi/server.go`
- `mobile_app/Makefile`
- `mobile_app/pubspec.yaml`
- `mobile_app/lib/main.dart`
- `mobile_app/test/widget_test.dart`

## Expected Next Step

- `make run-dev` bilan hammasini ko'tarish.
- Android emulator ichida appdan `Connect` bosib state olish.
- Agar ulanishda xato bo'lsa, to'g'ridan-to'g'ri network/emulator path fix qilish.
