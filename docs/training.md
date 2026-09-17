# ฝึกใช้ก่อนได้เครื่องจริง

ดัชนีเช้าวันนี้: [getting-started.md](getting-started.md)

ใช้ตัวช่วยโหมดฝึกซ้อม (`-synthetic`) กับตัวอย่างคลินิก
**ไม่ต้องรอเครื่องอ่าน ACR39U** บัตรเป็นข้อมูลจำลอง ไม่ใช่ผู้ป่วยจริง

ครั้งแรกยังต้องกด **อนุญาต** บนหน้า พร้อม — หน้าคลินิกอนุมัติตัวเองไม่ได้

ห้ามส่งภาพหน้าจอ เลขบัตร ชื่อ ที่อยู่ หรือรูปถ่ายไปในทิคเก็ต
ส่งได้เฉพาะ Copy diagnostics, `openthaiid audit` JSON (รหัสเท่านั้น)
และรหัสผิดพลาด เช่น `NO_CARD`

## สั้น ๆ

ซ้อมก่อนได้เครื่อง (เส้นทางซอฟต์แวร์เช้าวันศุกร์บนบัตรจำลอง — ไม่ใช่ผ่าน ACR39U):

```bash
make friday-dry-run
# ซ้อมเลือกเครื่องอ่าน (สองเครื่องจำลอง):
./scripts/friday-dry-run.sh --two-readers
# Windows: powershell -ExecutionPolicy Bypass -File scripts/friday-dry-run.ps1
```

คำสั่งนี้สร้างตัวช่วยถ้ายังไม่มี เปิด `-synthetic` รัน qualify-smoke ตรวจ
หน้า พร้อม / `GET /v1/helper/summary` / ทดสอบการเชื่อมต่อ (`bridge.probe`) /
`openthaiid doctor`
แล้วพิมพ์สรุปผ่าน ไทย/อังกฤษ ชี้ไปที่ [getting-started.md](getting-started.md)
กับ [friday-checklist.md](friday-checklist.md) — ไม่พิมพ์เลขบัตร ชื่อ หรือรูป

1. เปิดตัวช่วยฝึกซ้อมเอง (ถ้ายังไม่ได้รัน dry-run):

   ```bash
   make build
   ./dist/openthaiid -synthetic
   ```

   หน้า Ready (`http://127.0.0.1:17891/`) ต้องขึ้น **พร้อม (บัตรฝึกซ้อม)**
   และมีกล่อง **ฝึกใช้ก่อนได้เครื่องจริง**
   กด **ทดสอบการเชื่อมต่อ** ได้ — ต้องบอกว่าการเชื่อมต่อปกติ และไม่อ่านข้อมูลบนบัตร

2. เปิดตัวอย่างคลินิก (อีกเทอร์มินัล):

   ```bash
   cd sdk/web && npm ci && npm run build && cd ../..
   python3 -m http.server 8000
   ```

   เปิด `http://127.0.0.1:8000/examples/clinic-patient/`

3. กด **กรอกข้อมูลจากบัตรประชาชน**
4. ที่หน้า พร้อม กด **อนุญาต** (ให้รหัสตรงกันถ้ามี)
5. ช่องชื่อ / วันเกิดมีข้อมูลจำลอง — นั่นคือฝึกสำเร็จ

ถ้าไฟสถานะขึ้น **ยังไม่พร้อม** ให้ดูตารางใน [user.md](user.md)
ข้อความผิดพลาดบนหน้าคลินิกเป็นภาษาไทยเป็นค่าเริ่มต้น (`?lang=en` สำหรับอังกฤษ)
รหัสโปรโตคอลไม่เปลี่ยน

ถ้าตัวช่วยหลุดหรือเครื่องหลับ/ตื่น หน้าคลินิกต้องบอก `BRIDGE_UNAVAILABLE`
หรือ `PCSC_UNAVAILABLE` — ไม่ต้องรีสตาร์ทเครื่อง กด **ทดสอบการเชื่อมต่อ**
แล้วรอ **พร้อม** (ช่างเทคนิคกด **Reconnect card service** ได้ถ้าบริการอ่านบัตรดับ)
โหมดฝึกซ้อมจำลองบริการอ่านบัตรดับในชุดทดสอบ ไม่จำลอง sleep ของเครื่อง
วันศุกร์ E5/E6: [friday-checklist.md](friday-checklist.md) ·
ช่างเทคนิค: [technician.md](technician.md)

ถ้าถอดบัตรหรือถอดเครื่องอ่านระหว่างกรอก หน้าคลินิกต้องบอกว่าเกิดอะไร
(`CARD_REMOVED` / `READER_REMOVED`) ให้เสียบใหม่ รอ **พร้อม** แล้วกด กรอกข้อมูล
อีกครั้ง — อย่าใช้ค่าในช่องถ้ากรอกไม่สำเร็จ
โหมดฝึกซ้อมไม่มีเครื่องจริงให้ถอด แต่ชุดทดสอบสังเคราะห์ครอบคลุมเส้นนี้แล้ว
วันศุกร์ E4: [friday-checklist.md](friday-checklist.md) ·
เจ้าหน้าที่: [user.md](user.md)

## เมื่อได้เครื่องจริง

ปิดตัวช่วยฝึกซ้อม แล้วเปิดใหม่ **โดยไม่ใส่** `-synthetic`
พิมพ์ [friday-desk-card.md](friday-desk-card.md) ไว้ที่โต๊ะเจ้าหน้าที่
เสียบ ACR39U ตามบัตรนั้น (ไม่ใช่ชุด qualify)
ชุดช่างเทคนิค: [friday-checklist.md](friday-checklist.md)
อย่าสับสนบัตรจำลองกับบัตรผู้ป่วย

รายละเอียดช่างเทคนิค: [technician.md](technician.md)

---

# Practice before the real reader arrives (English)

Start with `make friday-dry-run` (ซ้อมก่อนได้เครื่อง) to rehearse the
Friday-morning software path on `-synthetic` without an ACR39U. Then start
`./dist/openthaiid -synthetic`, open
`examples/clinic-patient/`, click Fill, then **Allow** on the Ready page.
The form shows fictional card data. Do not send screenshots or field
values in tickets — diagnostics, `openthaiid audit` JSON, and error
**codes** only.
Pass `{ lang: "en" }` / `?lang=en` for English staff banners.
When the ACR39U arrives, restart the helper without `-synthetic`.
Friday-morning index: [getting-started.md](getting-started.md).
Staff desk card (print one Thai page): [friday-desk-card.md](friday-desk-card.md).
Friday one-page (multi-reader, E4–E6): [friday-checklist.md](friday-checklist.md).
Technicians: [technician.md](technician.md).
Loopback HTTP inventory: [security.md](security.md).
Firefox/Safari https Fill: [compatibility.md](compatibility.md) · optional
helper TLS: [loopback-tls.md](loopback-tls.md).
