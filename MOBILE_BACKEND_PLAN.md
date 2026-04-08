# Mobile Backend Plan

## Maqsad

`gscale-zebra` uchun mobile app bilan ishlaydigan, lokal tarmoqda realtime holat va boshqaruvni beradigan backend qatlamini bosqichma-bosqich qurish.

Asosiy prinsip:

- mavjud `scale`, `bot`, `bridge`, `polygon` oqimlarini buzmaslik
- backendni `mobileapi` orqali alohida qatlam sifatida chiqarish
- avval backend contractni mustahkamlash, keyin Flutter UI ni ulash
- realtime monitoring va action oqimlarini bir xil API modelga tushirish

## Hozirgi Holat

Tayyor qismlar:

- `bridge_state.json` markaziy truth-source
- `polygon` orqali real qurilmasiz test muhiti
- `mobileapi`ning boshlang'ich versiyasi

Hozir ishlayotgan endpointlar:

- `GET /healthz`
- `POST /v1/mobile/auth/login`
- `POST /v1/mobile/auth/logout`
- `GET /v1/mobile/profile`
- `GET /v1/mobile/monitor/state`

## Yakuniy Yo'nalish

Arxitektura:

- Mini PC:
  - `scale`
  - `bot`
  - `bridge`
  - `mobileapi`
  - `polygon` dev/test uchun
- Mobile app:
  - `mobile_app` Flutter client
- Aloqa:
  - bir xil Wi-Fi ichida lokal IP orqali
  - realtime uchun `WebSocket` yoki `SSE`
  - action uchun `HTTP POST`

## Bosqichlar

### 1. Backend Foundation

Maqsad:

- `mobileapi`ni stabil minimal backendga aylantirish

Ishlar:

- `healthz`
- auth/login/profile
- monitor snapshot
- `bridge_state`dan o'qish
- `polygon` trace bilan integratsiya

Holat:

- bajarilgan

### 2. Realtime Stream

Maqsad:

- mobile app polling qilmasdan live holat olsin

Ishlar:

- `WS /v1/mobile/monitor/stream` yoki `SSE /v1/mobile/monitor/stream`
- event modeli
- change detection
- reconnect strategiya

Eventlar:

- `scale_changed`
- `zebra_changed`
- `batch_changed`
- `print_request_changed`
- `printer_command_changed`
- `service_status`

Acceptance:

- scale o'zgarishi 1 soniyadan kam ichida appga yetib boradi
- reconnectdan keyin current snapshot qayta olinadi
- polygon bilan dev sinov ishlaydi

### 3. Action API

Maqsad:

- appdan PC’ga boshqaruv yuborish

Birinchi actionlar:

- `POST /v1/mobile/actions/read-rfid`
- `POST /v1/mobile/actions/encode`
- `POST /v1/mobile/actions/batch-start`
- `POST /v1/mobile/actions/batch-stop`
- `POST /v1/mobile/actions/dev/weight`
- `POST /v1/mobile/actions/dev/reset`

Keyingi actionlar:

- `POST /v1/mobile/actions/calibrate`
- `POST /v1/mobile/actions/restart-service`
- `GET /v1/mobile/logs`

Acceptance:

- action javobi aniq status qaytarsin
- action natijasi realtime stream orqali ham aks etsin
- polygon bilan action test qilinsa trace ko'rinsin

### 4. Local Network Discovery

Maqsad:

- foydalanuvchi IP yozib o'tirmasin

Variantlar:

- MVP:
  - qo'lda IP kiritish
- Next:
  - `mDNS/Bonjour`
  - `gscale.local`
- Optional:
  - QR pairing

Acceptance:

- bir xil Wi-Fi ichida server topilishi yoki qo'lda saqlanishi
- app local backendga ulana olishi

### 5. Flutter Integration

Maqsad:

- mavjud `mobile_app` UI bazasini bizning backendga ulash

Birinchi ekranlar:

- login
- monitor dashboard
- printer trace
- simple controls
- settings/server connection

Saqlanadigan qismlar:

- auth UI
- app shell
- theme
- session
- navigation

Moslashtiriladigan qismlar:

- `MobileApi.baseUrl`
- login contract
- monitor models
- role routing

Keyinga qoldiriladigan qismlar:

- eski supplier/werka/customer/admin business oqimlari

Acceptance:

- login ishlashi
- realtime dashboard ishlashi
- kamida 2 ta action appdan yuborilishi

### 6. Real Device Validation

Maqsad:

- real scale va real Zebra bilan end-to-end tekshiruv

Ishlar:

- polygon o'rniga real source ulash
- timing, busy, permission xatolarni ko'rish
- batch oqimini real uskunada verifikatsiya qilish

Acceptance:

- realtime app real qurilma bilan ham ishlashi
- actionlar real printer/scale bilan mos ishlashi

## MVP Chegarasi

MVP uchun yetarli:

- login
- monitor snapshot
- realtime stream
- printer trace
- batch status
- 2-4 ta action
- local server configuration

MVPdan tashqarida:

- barcha eski role flowlarini ko'chirish
- murakkab ERP mobil workflowlar
- push notification integratsiyasini to'liq moslashtirish

## Texnik Qarorlar

### Realtime

Tavsiya:

- birinchi versiyada `SSE`

Sabab:

- server tomoni sodda
- mobile monitor uchun yetarli
- reconnect oddiy

Keyin kerak bo'lsa:

- `WebSocket`

### Auth

Birinchi versiya:

- local dev login

Keyin:

- pairing token
- device-bound session

### Source of Truth

Asosiy qoida:

- mobile app hech qachon truth-source bo'lmaydi
- truth-source:
  - `bridge_state`
  - real worker state

## Hozirgi Eng Yaqin Ishlar

1. realtime stream endpoint qo'shish
2. stream event modelini aniqlash
3. action API ning birinchi 2-3 endpointini qo'shish
4. `mobile_app`da monitor dashboardni ulash

## Ish Tartibi

Har bosqich uchun:

1. kod yozish
2. lokal test
3. `polygon` bilan tekshirish
4. commit qilish

## Eslatma

Bu reja backend-first yondashuv uchun yozildi. UI keyingi bosqichlarda tezroq yurishi uchun backend contractni oldin muzlatish kerak.
