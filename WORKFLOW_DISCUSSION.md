# Workflow Discussion Notes

Bu fayl loyiha workflow, business semantika, va arxitektura boyicha muhokama natijalarini jamlash uchun yozildi.
Hozircha bu yerda kod ozgartirish emas, qarorlarni aniqlashtirish maqsad qilingan.
Kod darajasidagi implementatsiya keyin, barcha muhim masalalar yopilgandan song qilinadi.

## 1. Muhokamaning maqsadi

Bu hujjatning vazifasi:

- Hozirgi workflowdagi mos tushmayotgan joylarni aniqlash.
- Business oqimga togri semantika tanlash.
- Keyin kodga tushiriladigan yagona modelni oldindan kelishib olish.
- Ortiqcha abstraction yoki ikki xil manoli identitylardan qochish.

## 2. Real biznes oqimi

Muhokama davomida real jarayon quyidagicha ekanligi aniqlandi:

- Mahsulot zavodda tayyor boladi.
- Mahsulot tarozida tortiladi.
- Mahsulotga RFID yopishtiriladi.
- Keyin mahsulot omborga yuboriladi.

Muhim nuqta:

- Bu ombordan chiqish jarayoni emas.
- Bu omborga kirish yoki qabul qilishga yaqin jarayon.
- Shuning uchun ERP hujjat turini togri tanlash muhim.

## 3. Asosiy biznes invariantlar

Muhokama davomida eng muhim va qat'iy qoidalar sifatida quyidagilar qabul qilindi:

### 3.1. Bitta line modeli

- Hozirgi tizim bitta fizik line uchun ishlaydi.
- Bitta tarozi.
- Bitta Zebra printer.
- Bitta real ishlab chiqarish oqimi.
- Shu sabab parallel mustaqil oqimlar hozircha kozda tutilmaydi.

### 3.2. Bitta aktiv batch global boladi

- `single active batch globally`
- Bir vaqtning ozida bir nechta mustaqil batch session ochilishi notogri deb topildi.
- Sabab: fizik line bitta bolgani uchun tizimda ham bir vaqtning ozida bitta haqiqiy aktiv batch bolishi kerak.
- Aks holda `bridge_state.json` kabi global state bilan bot session modeli bir-biriga mos tushmaydi.

### 3.3. Yagona identity qoidasi

Muhokamadagi eng muhim qarorlardan biri:

- `1 EPC = 1 cycle = 1 ERP Stock Entry`

Bu qoidaning manosi:

- Bitta stable yakuniy natija uchun bitta RFID identity boladi.
- Osha identity printer/tag ichida ham, ERP yozuvida ham aynan bitta narsani anglatadi.
- Bitta natija uchun bir nechta har xil unique ID ishlatish xato deb qabul qilindi.

Muhim aniqlashtirish:

- `cycle_id` degani alohida uchinchi yangi qiymat emas.
- `cycle_id` bu faqat workflow tili.
- Real unique qiymat sifatida `EPC`ning ozi qabul qilindi.

Demak:

- Printerga yoziladigan ham `EPC`
- ERPga saqlanadigan ham `EPC`
- Workflow ichida cycle identity sifatida qaraladigan ham `EPC`

### 3.4. Bir harakatga bitta RFID

Qat'iy business qoida sifatida:

- Bitta harakatga bitta RFID tegishli boladi.
- Ikki yoki undan ortiq unique IDni bitta natijaga boglash business jihatdan ham, arxitektura jihatdan ham xato.
- `1 EPC = 1 cycle = 1 ERP row` qoidasi shu nuqtadan kelib chiqqan.

## 4. ERP hujjat semantikasi

### 4.1. Hozirgi holat

Hozirgi kod:

- `Stock Entry` yaratadi
- `stock_entry_type = Material Issue` ishlatadi

### 4.2. Nega bu noto'g'ri deb topildi

Muhokama davomida quyidagisi aniqlandi:

- Jarayon ombordan chiqish emas
- Mahsulot zavodda tortilyapti
- RFID yopishtirilyapti
- Keyin omborga jonatilyapti

Shu sabab `Material Issue` business semantika boyicha togri emas deb topildi.

### 4.3. Yangi kelishuv

Hozirgi business oqim uchun togriroq semantika:

- `Stock Entry`
- `Stock Entry Type = Material Receipt`

Muhim eslatma:

- Bu `Material Request` emas.
- Bu hanuz `Stock Entry` familyasidagi hujjat.
- Faqat `Issue` emas, `Receipt` business manoga mosroq.

## 5. ERP ichida EPC qayerga yoziladi

Bu mavzu muhokama davomida alohida aniqlashtirildi.

Avval variant sifatida:

- `serial_no`
- `barcode`
- yoki ikkalasi birga

Muhokama natijasi:

- ERP ichida canonical maydon sifatida faqat `barcode` ishlatiladi.
- `barcode = EPC`
- `serial_no` ishlatilmaydi.

Bu qarorning sababi:

- Bitta identity bitta maydonda yashashi kerak.
- Bir xil EPCni bir vaqtning ozida ikki fieldga yozish ortiqcha duplication.
- ERP ichida ham identity bitta maydonda saqlanishi kerak.

Yakuniy kelishuv:

- `barcode = EPC`
- `serial_no`ga tegilmaydi

## 6. EPC format boyicha kelishuv

Muhokama davomida EPC formatini ozgartirish kerakmi degan savol ham korildi.

Hozirgi EPC:

- `24` belgilik
- uppercase `HEX`
- vaqt, ketma-ketlik, va salt aralashmasi bilan generatsiya qilinadi

Qaror:

- Hozirgi format o'z holicha qoldiriladi.
- Prefiks, yangi pattern, yoki inson kozi uchun boshqa shakl kiritilmaydi.
- Hozirgi `24-char hex EPC` biznes uchun ham, texnik nuqtai nazardan ham yetarli deb topildi.

Muhim eslatma:

- Uniqueness masalasi keyin alohida ozgartirilmaydi.
- Hozirgi EPC generatorning o'zi uniq bolishini taminlaydigan asosiy manba sifatida qabul qilindi.

## 7. Cycle va print logikasi

Bu eng muhim muhokamalardan biri boldi.

### 7.1. Noto'g'ri deb topilgan variant

Avval `yangi cycle faqat tarozi 0 ga tushganda ochilsin` degan variant korildi.
Bu variant business jihatdan notogri deb topildi.

Nega notogri:

- Tare, babina, yoki packaging vazni tarozida qolishi mumkin.
- Mahsulot sof vazni emas, yakuniy package vazni kerak bolishi mumkin.
- Bir nechta mahsulot bitta o'ramga yigilib, keyin bitta RFID bilan belgilanishi mumkin.
- Keyingi cycle oldingisidan og'ir bolishi ham, yengil bolishi ham, hatto deyarli teng bolishi ham mumkin.

Shu sabab `0` ga tushishni majburiy shart qilish kritik xato deb topildi.

### 7.2. To'g'ri deb qabul qilingan model

Qabul qilingan model:

- `stable -> print`
- `movement -> next stable -> next print`

Demak:

- Mahsulot qoyiladi
- Tarozi stable boladi
- Shu stable qty qabul qilinadi
- Shu stable qty uchun EPC yaratiladi va print ketadi
- Keyin tarozida yana harakat bo'ladi
- Keyin yana stable nuqta hosil boladi
- Shu yangi stable nuqta yangi cycle hisoblanadi

Muhim aniqlashtirish:

- Yangi stable oldingi stable bilan bir xil bolishi ham mumkin
- Og'irroq bolishi ham mumkin
- Yengilroq bolishi ham mumkin
- `0` bolishi ham mumkin

Lekin bularning hech biri majburiy shart emas.

Majburiy shart bitta:

- Oldingi stable event bilan keyingi stable event orasida haqiqiy movement phase bolishi kerak.

### 7.3. Movement qayerdan aniqlanadi

Bu ham muhokama davomida aniqlandi:

- Movement yoki stable holatni dastur ozidan chiqarmaydi
- Tarozining ozidan kelayotgan stable/unstable logika asosiy manba bo'ladi

Demak:

- Yangi cycle `weight delta` bo'yicha emas
- Yangi cycle `tarozi harakat qildi -> yana stable boldi` mantig'i bilan aniqlanadi

## 8. Hozir aniqlangan arxitektura muammolari

Muhokama davomida quyidagi mos kelmasliklar topildi.

### 8.1. Bot session modeli va global state modeli bir-biriga mos emas

- Bot tarafida bir nechta session saqlanishi mumkin
- Lekin `bridge_state.json` global va bitta batch holatni saqlaydi
- Bitta fizik line uchun bu xavfli
- Shu sabab `single active batch globally` modeli qabul qilindi

### 8.2. ERP xatoda duplicate xavfi mavjud

- Agar ERP draft yaratish muvaffaqiyatsiz bolsa
- Shu cycle qayta-qayta urilishi mumkin
- Bu duplicate draft yoki duplicate harakat xavfini keltirib chiqaradi

### 8.3. Verify muvaffaqiyatsiz bolsa ham downstream davom etish xavfi mavjud

- Hozirgi dizayn RFID verify masalasida yumshoq bo'lishi mumkin
- Bu esa printer/tag va ERP orasida nomuvofiqlik keltirib chiqarishi mumkin

Keyingi muhokamada bu xavf yumshatildi:

- `verify` yakuniy gate sifatida ishlatilmaydi
- worker success mezoni printer command va printer statusiga bog'lanadi
- print muvaffaqiyatsiz bo'lsa draft delete qilinadi

### 8.4. Identity modeli avval chalkash edi

Muhokamadan oldin:

- EPC
- cycle
- ERP row identity

bir-biridan alohida narsadek korinishi mumkin edi.

Muhokama natijasida bu soddalashtirildi:

- Bular aslida bitta identityning uch xil ko'rinishi emas
- Real identity bitta: `EPC`
- `cycle` bu processdagi nom
- ERP yozuvi esa shu EPCni aks ettiradi

### 8.5. Hozirgi oqim teskari qurilgan bo'lishi mumkin

Muhokama davomida juda muhim arxitektura nuqsoni aniqlashtirildi:

Hozirgi oqim taxminan:

- scale stable bo'ladi
- EPC generatsiya qilinadi
- printerga encode/print ketadi
- keyin bot shu EPCni olib ERPga yozadi

Bu yondashuv noto'g'ri deb topildi.

Sabab:

- Agar ERP hali qabul qilmagan bo'lsa, printer/tag ichida identity allaqachon fizik dunyoga chiqib ketadi.
- Keyin ERP qismi yiqilsa, identity bilan ERP yozuvi orasida ajralish paydo bo'ladi.
- Bu `1 EPC = 1 cycle = 1 ERP Stock Entry` qoidaga zid xavf tug'diradi.

Muhokama natijasi:

- `ERP-first` oqimi tog'riroq deb topildi.

Yangi maqsadli oqim:

- scale stable holatni aniqlaydi
- bot shu cycle uchun EPC bilan ERP `Material Receipt` yaratadi
- ERP muvaffaqiyatli bo'lsa, aynan shu EPC printerga yuboriladi
- printer natijasi yana workflowga qaytadi

Muhim qoidasi:

- EPC candidate sifatida yaratiladi
- Agar ERP duplicate check bosqichida shu candidate EPC band ekani aniqlansa, yangi candidate EPC olinishi mumkin
- ERP draft muvaffaqiyatli yaratilganidan keyin esa shu cycle uchun EPC muzlaydi
- Shundan keyin retry bo'lsa ham o'sha final EPC ishlatiladi

### 8.6. ERP xatoda yangi EPC yaratish noto'g'ri

Muhokamada quyidagi variant ko'rildi:

- Agar ERP bilan muammo bo'lsa, yangi EPC generatsiya qilinib yuborilsinmi

Qaror:

- Bu savol ikki holatga ajratildi

1. ERP duplicate check paytida shu EPC allaqachon mavjud ekani aniqlansa
2. ERP draft allaqachon yaratilgan bo'lsa yoki cycle final bosqichga o'tgan bo'lsa

Sabab:

- Duplicate check paytida hali final EPC tanlanmagan bo'ladi
- Final EPC printerga ketadigan va ERPga birikadigan EPC hisoblanadi

Yakuniy qoidasi:

- Agar duplicate printdan oldin, ERP qabul qilish bosqichida aniqlansa, shu in-progress cycle uchun darhol yangi EPC generatsiya qilinadi
- Yangi EPC bilan qayta urinish qilinadi
- Printerga faqat ERP muvaffaqiyatli qabul qilgan EPC yuboriladi
- ERP draft yaratilganidan keyin esa shu cycle uchun EPC muzlaydi va endi qayta generatsiya qilinmaydi

## 9. Hozircha ataylab keyinga qoldirilgan mavzular

Ba'zi masalalar hozir muhokama qilindi, lekin ataylab implementatsiyaga olib kirilmadi.

### 9.1. Verify policy

Avval quyidagi variantlar ko'rildi:

- `verify` qat'iy bloklaydimi
- yoki warning bilan davom etadimi
- yoki operator tasdiqini kutadimi

Muhokama davomida bu masalani soddaroq yopadigan kuchli yondashuv taklif qilindi:

- Avval ERP ichida draft yaratiladi
- Keyin print qilinadi
- Agar print worker tomonidan xatolik bilan tugasa, draft delete qilinadi
- Agar print muvaffaqiyatli tugasa, draft submit qilinadi

Bu yondashuvning foydasi:

- ERP va print o'rtasida aniq ikki bosqichli state paydo bo'ladi
- Print muvaffaqiyatsiz bo'lsa ERP ichida yakuniy noto'g'ri hujjat qoldirilmaydi
- Print muvaffaqiyatli bo'lsa yakuniy submit qilinadi
- Business semantika juda sodda bo'lib qoladi

Muhim eslatma:

- Bu model ishlashi uchun `print success` worker tomonidan ishonchli tarzda qaytarilishi kerak
- Faqat command yuborildi degan signal yetarli emas
- Agar media out, no tag, yoki shunga o'xshash printer xatosi worker tomonidan ushlansa, draft delete qilinadi

Shu sabab `verify` mavzusi alohida policy bo'lib qolayotgan bo'lsa ham, asosiy workflow endi `draft -> print -> submit/delete` mantig'i bilan soddalashtirilishi mumkin.

Keyingi muhokama natijasida `verify` boyicha yana muhim aniqlashtirish qilindi:

- `verify` gate sifatida ishlatilmaydi
- Sabab: amaliy testlarda printer ba'zida yozilgan RFIDni qayta o'qib tekshirishda noto'g'ri yoki ishonchsiz natija berishi mumkin
- Yani printer yozgan bo'lsa ham, keyingi check paytida "yozilmadi" yoki noaniq ko'rinishi ehtimoli bor

Yakuniy amaliy qaror:

- Worker uchun `print success` mezoni:
- command yuborildi
- printer xato qaytarmadi
- printer statusi normal qoldi

Demak:

- `verify` foydali diagnostik signal bo'lib qoladi
- Lekin u yakuniy success/fail gate bo'lmaydi
- `verify` yomon chiqdi degani avtomatik `print fail` degani emas

### 9.2. ERP duplicate himoya

Muhokama natijasi:

- Agar shu `barcode = EPC` bilan ERPda yozuv allaqachon mavjud bo'lsa, current in-progress cycle to'xtatilmaydi
- Shu zahoti yangi candidate EPC generatsiya qilinadi
- Yangi EPC bilan ERP create qayta uriniladi
- ERP muvaffaqiyatli qabul qilgan EPCgina printerga yuboriladi

Demak duplicate himoya:

- stop emas
- skip emas
- yangi candidate EPC bilan davom etish

### 9.3. ERP failure policy

Avval quyidagi variantlar ko'rildi:

- ERP create xato bersa line nima qilsin
- Retry
- Pause
- Skip
- Manual aralashuv

Muhokama davomida bu masala soddalashtirildi:

- Bot ERP bilan admin darajadagi API orqali ishlashi kerak
- Permission muammolari design driver sifatida qaralmaydi
- Custom scriptlarga tayanilmaydi, imkon qadar rasmiy va sodda flow ishlatiladi
- Server yoki ERP ishlamay qolsa, batch `pause` holatga o'tadi
- Shu holat uchun yangi print, yangi submit, yoki yangi cycle davom ettirilmaydi
- Operator keyin batchni qayta davom ettiradi yoki qayta start qiladi

Muhim amaliy qaror:

- `print fail` asosiy rollback trigger bo'lib qoladi
- `server/ERP down` bo'lsa esa `batch pause` ishlaydi
- `submit fail` kam uchraydigan exception sifatida qaraladi, asosiy design markazi emas
- `batch pause` bo'lsa current in-flight cycle davom ettirilmaydi
- Resume bo'lganda tizim eski cyclega qaytmaydi
- Resume bo'lganda tizim faqat keyingi yangi cycle bilan davom etadi

### 9.4. EPC uniqueness implementation detali

Muhokamada uniqueness mavzusi kotarildi.

Kelishuv:

- EPC unique bolishi kerak
- Hozirgi generator bu vazifani bajaradi
- Formatni ozgartirishga hozircha ehtiyoj yoq
- Bu masala keyingi implementatsiyada qayta ochilmaydi, agar real muammo chiqmasa

### 9.5. Draft, delete va submit semantikasi

Yangi workflow g'oyasiga ko'ra:

- ERP avval `draft` yaratadi
- Print muvaffaqiyatli bo'lsa `submit`
- Print muvaffaqiyatsiz bo'lsa `delete`

Bu ayniqsa quyidagi ssenariy uchun foydali deb topildi:

- ERP draft muvaffaqiyatli yaratildi
- Lekin label tugab qoldi yoki printer xato berdi
- Bu holda draftni delete qilib yuborish mumkin

Shu bilan ERP ichida noto'g'ri yoki yetim yakuniy harakat qoldirmaslikka erishiladi

## 10. Arxitektura soddaligi boyicha kelishuv

Muhokama davomida quyidagi prinsip alohida qayd qilindi:

- Minimal ish bilan maksimal natija olish kerak
- Arxitektura ortiqcha murakkablashmasligi kerak
- Tizim shunchalik sodda bo'lishi kerakki, uni oddiy odamga ham tushuntirish mumkin bo'lsin

Shu nuqtadan qaralganda qabul qilingan model juda sodda:

- Bot business qaror beradi
- Worker hardware ishini bajaradi
- Bridge ular orasidagi kanal bo'ladi

Bu modelning sodda ko'rinishi:

- stable bo'ldi
- ERP yaratildi
- print buyrug'i berildi
- printer yozdi

Bu minimal o'zgarish bilan maksimal semantik to'g'rilikka olib keladigan yo'l deb baholandi.

## 11. Bot, worker va bridge o'rtasidagi yangi rol taqsimoti

Muhokama davomida quyidagi arxitektura yo'nalishi eng tog'ri deb topildi:

### 11.1. Botning roli

- batch workflow boshqaruvi
- ERP bilan ishlash
- qaysi EPC qabul qilinganini aniqlash
- qachon print qilish kerakligini hal qilish

### 11.2. Worker's roli

- tarozi bilan ishlash
- Zebra printer bilan ishlash
- encode/print bajarish
- verify yoki printer holatini qaytarish

### 11.3. Bridge'ning roli

- bot va worker orasidagi command/status kanali bo'lish
- faqat latest snapshot emas, balki workflow buyruqlarini ham tashish

Muhokama natijasi:

- printer bilan to'g'ridan-to'g'ri bot ishlamaydi
- printer buyrug'ini worker bajaradi
- bot printni bridge orqali workerga buyuradi

### 11.4. Nega bu model tanlandi

Bu model:

- hardware logikani worker ichida qoldiradi
- business orchestrationni bot ichida qoldiradi
- `ERP-first` qoidani saqlaydi
- ortiqcha yangi servis yoki murakkab channel talab qilmaydi
- mavjud arxitekturani eng kam o'zgarish bilan to'g'ri tomonga buradi

### 11.5. Muhim amaliy oqim

Muhokama asosida maqsadli oqim quyidagicha bo'lishi kerak:

- scale stable holatni aniqlaydi
- shu cycle uchun EPC tayyor bo'ladi
- bot ERP `Material Receipt` yaratadi
- ERP muvaffaqiyatli bo'lsa, bot bridge orqali print buyrug'ini workerga beradi
- worker aynan shu EPCni printerga yozadi
- natija status sifatida qaytadi

### 11.6. Bridge ichida print buyrug'ini ifodalash boyicha tavsiya

Muhokama davomida bridge ichidagi print buyruq formati ham korib chiqildi.

Korilgan variantlar:

- faqat `pending_print_epc` kabi oddiy field
- alohida kichik `print_request` object

Tavsiya etilgan variant:

- `single print_request object`

Sabab:

- Bu hali ham juda sodda
- Lekin keyin `pending/done/error` holatlarini saqlash mumkin
- Retryni shu object ichida boshqarish oson
- Workerga print uchun kerakli malumotlarni bitta joyda berish mumkin

Kutilayotgan semantika:

- bot `print_request.status = pending` qiladi
- worker shu requestni oladi
- print bajarilgach `done` yoki `error` qaytaradi
- shu bilan bot va worker orasidagi command/status zanjiri sodda, lekin boshqariladigan bo'ladi

Muhim eslatma:

- Bu object yangi business identity yaratmaydi
- Asosiy identity hanuz `EPC`ning ozi bo'lib qoladi
## 12. Hozircha yopilgan mavzular

Quyidagilar muhokamada yopildi:

- Loyiha `single line`
- Batch modeli `single active batch globally`
- `1 EPC = 1 cycle = 1 ERP Stock Entry`
- Real unique identity `EPC`
- ERP ichida `barcode = EPC`
- `serial_no` ishlatilmaydi
- ERP semantikasi `Material Issue` emas, `Material Receipt`
- EPC formati hozirgi `24-char hex` ko'rinishida qoladi
- `0` ga tushishni kutish kerak emas
- Yangi cycle `movement -> stable` mantigi bilan ochiladi
- Stable/unstable holati tarozining o'zidan olinadi
- `ERP-first` workflow tog'riroq
- ERP duplicate check bosqichida yangi candidate EPC olish mumkin
- ERP draft yaratilganidan keyin shu cycle uchun EPC qayta generatsiya qilinmaydi
- Printni bot emas, worker bajaradi
- Bot printni bridge orqali buyuradi
- Tavsiya etilgan target workflow: `draft -> print -> success bo'lsa submit, fail bo'lsa delete`

## 13. Hozircha ochiq qolgan savollar

Hozircha katta semantik yoki arxitektura darajasidagi ochiq savol qolmadi.

Keyingi bosqich:

- Shu hujjat asosida implementatsiya planini chiqarish
- Keyin kod darajasida bosqichma-bosqich ozgartirish kiritish

## 14. Hozirgi ish qoidasi

- Hozircha kod ozgartirilmaydi
- Avval barcha semantik va arxitektura qarorlari yakunlanadi
- Keyin shu fayldagi kelishuvlarga tayangan holda implementatsiya boshlanadi
