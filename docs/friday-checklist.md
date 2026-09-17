# วันศุกร์ — ACR39U (หนึ่งหน้า)

ดัชนีเช้าวันนี้: [getting-started.md](getting-started.md)  
บัตรโต๊ะเจ้าหน้าที่ (พิมพ์หนึ่งหน้า ไทย): [friday-desk-card.md](friday-desk-card.md)

เสียบเครื่องอ่าน → พร้อม → qualify → กรอกข้อมูล → ส่งให้ jarvis เฉพาะวินิจฉัย
**ห้าม** ส่งเลขบัตร ชื่อ ที่อยู่ วันเกิด รูปถ่าย ภาพหน้าจอฟอร์ม หรือ HAR ที่มียอด `card.read`

ฝึกซ้อมโดยไม่มีเครื่อง: [training.md](training.md) (`make friday-dry-run` /
`-synthetic`) — ซ้อมก่อนได้เครื่อง ไม่ใช่ผ่าน ACR39U
ชุดเต็ม: [hardware-qualification.md](hardware-qualification.md)
เว็บคลินิกปิดตัวช่วยไม่ได้: [security.md](security.md)
Firefox / Safari กรอกจากหน้าคลินิก **HTTPS**: [compatibility.md](compatibility.md)
· HTTPS ของตัวช่วย: [loopback-tls.md](loopback-tls.md)

## 1. เสียบเครื่องอ่าน → พร้อม

- เสียบ **ACR39U** เข้าเครื่องนี้ เปิด OpenThaiID (**ไม่ใช้** `-synthetic`)
  ถ้าแผนก IT ติดตั้ง always-on แล้ว (`openthaiid service install` /
  `Install-OpenThaiID-Service.ps1`) ไม่ต้องเปิดถาด — ดู
  [installation.md](installation.md)
- `openthaiid version` และ `openthaiid doctor` — helper running, `pcsc: OK`,
  ชื่อเครื่องอ่าน, `personal fields: not read` (ไม่มี path ของกุญแจ TLS)
- หน้า Ready `http://127.0.0.1:17891/` ขึ้น **พร้อม** หลังเสียบบัตรประชาชน
- ถ้าเสียบเครื่องอ่านมากกว่าหนึ่งเครื่อง ให้กด **ใช้เครื่องนี้** บนหน้า พร้อม
  (หรือ Technician tools **Use this reader**) ก่อนกรอกข้อมูล —
  เจ้าหน้าที่: [user.md](user.md) · ช่างเทคนิค: [technician.md](technician.md)
  (Multi-reader desks)
- กด **ทดสอบการเชื่อมต่อ** บนหน้า พร้อม (ไม่อ่านข้อมูลบนบัตร)
- ถ้า **ยังไม่พร้อม**: เปิดตัวช่วย → เสียบเครื่องอ่านใหม่ → เสียบบัตรให้สุด
  อย่าส่งภาพบัตร
- ถ้าตัวช่วยหลุดระหว่างวัน: เปิดใหม่ หน้า พร้อม / หน้าคลินิกรอให้กลับมาได้เอง
  แล้วกด กรอกข้อมูล อีกครั้ง (ไม่อ่านบัตรซ้ำเอง)
- หลังเครื่องหลับ/ตื่น หรือบริการอ่านบัตรสะดุด: **ไม่ต้องรีสตาร์ทเครื่อง**
  กด **ทดสอบการเชื่อมต่อ** (ไม่อ่านข้อมูลบนบัตร) — ตารางเจ้าหน้าที่
  [user.md](user.md) · E5/E6 ช่างเทคนิค [technician.md](technician.md)

| ที่เห็น | รหัส | ปุ่ม | ทำต่อ |
|---|---|---|---|
| ตัวช่วยยังไม่ทำงาน / ทดสอบการเชื่อมต่อไม่สำเร็จ | `BRIDGE_UNAVAILABLE` | ทดสอบการเชื่อมต่อ | เปิด OpenThaiID เปิดหน้า พร้อมค้างไว้ รอไฟ แล้วกด กรอกข้อมูล อีกครั้ง |
| ตัวช่วยทำงาน แต่บริการอ่านบัตรยังไม่พร้อม | `PCSC_UNAVAILABLE` | ทดสอบการเชื่อมต่อ → Technician **Reconnect card service** | ไม่รีสตาร์ทเครื่อง รอไฟ **พร้อม** แล้วกรอกข้อมูล |
| บริการอ่านบัตรไม่ตอบสนอง | `BACKEND_STALLED` | **Reconnect card service** | เหมือนแถวบน |

ชุดทดสอบสังเคราะห์จำลองบริการอ่านบัตรดับแล้วกลับมาได้แล้ว
วันศุกร์บน ACR39U ยังต้องติ๊ก E5 / E6 ใน qualify (macOS อาจ SKIP E6 ถ้าหยุด PC/SC ไม่ได้)
รายละเอียดช่างเทคนิค: [technician.md](technician.md) Friday E5/E6

## 2. Qualify (ไม่เก็บข้อมูลบุคคล)

```bash
./dist/openthaiid qualify -operator -report ./openthaiid-qualify.json
# Windows package: Qualify-OpenThaiID.bat
```

พิมพ์ได้แค่ `pass` / `fail` / `skip` — ห้ามวางเลขบัตรหรือรูป
คำสั่งพิมพ์ `report: <absolute path>` หนึ่งไฟล์ JSON ที่ถูกกรองแล้ว
ตรวจว่าฟิลด์ `"pii"` บอกว่าไม่เก็บค่าบนบัตร

## 3. กรอกข้อมูล (Fill)

เปิด ERP หรือ `examples/clinic-patient/` → กด **กรอกข้อมูล** → กด **อนุญาต**
บนหน้า พร้อม ดูผลบนจออย่างเดียว **อย่าคัดลอกค่าในช่อง**

ถ้า Fill ไม่ขึ้น: ใช้ข้อความไทยบนหน้าคลินิก (รหัสคงที่ เช่น
`BRIDGE_UNAVAILABLE` `NO_READER` `NO_CARD` `CARD_REMOVED` `READER_REMOVED`
`PCSC_UNAVAILABLE` `HTTPS_WS_BLOCKED` `TLS_UNTRUSTED` `APPROVAL_DENIED`
`FIELD_NOT_ALLOWED`)
แล้วเปิด Technician tools สำหรับ **Operator audit trail** (รหัสเท่านั้น)
ดู [technician.md](technician.md)
Firefox / Safari บนหน้าคลินิก **https://**: [compatibility.md](compatibility.md)
หรือเปิด HTTPS ของตัวช่วยครั้งเดียวต่อเครื่อง ([loopback-tls.md](loopback-tls.md))
— หน้า พร้อม ที่ `http://127.0.0.1:17891/` ยังใช้ได้

ถอดบัตรหรือถอดเครื่องอ่านระหว่างกรอก:

| ที่เห็น | รหัส | ทำต่อ |
|---|---|---|
| บัตรถูกถอดระหว่างอ่าน | `CARD_REMOVED` | เสียบบัตรให้สุด รอไฟ **พร้อม** แล้วกด กรอกข้อมูล อีกครั้ง |
| เครื่องอ่านถูกถอดระหว่างอ่าน | `READER_REMOVED` | เสียบเครื่องอ่าน เสียบบัตร รอ **พร้อม** แล้วกด กรอกข้อมูล อีกครั้ง |
| อ่านไม่สำเร็จ / หมดเวลา | `READ_FAILED` / `READ_TIMEOUT` | เหมือนด้านบน หรือเรียกช่างเทคนิค |

อย่าใช้ค่าในช่องถ้ากรอกไม่สำเร็จ — `autofillThaiID` ล้างช่องที่ผูกกับการกรอก
(ไม่แตะเบอร์โทร / HN) การฝึกสังเคราะห์ครอบคลุมเส้นนี้แล้ว
วันศุกร์บน ACR39U ยังต้องติ๊ก E4 ใน qualify
เจ้าหน้าที่: [user.md](user.md) · ช่างเทคนิค E4: [technician.md](technician.md)

## 4. ส่งให้ jarvis (วินิจฉัยเท่านั้น)

ส่งได้:

- **Copy diagnostics** จากหน้า Technician tools (เวอร์ชัน, OS, บริการอ่านบัตร,
  ชื่อเครื่องอ่าน, presence, รหัสผิดพลาด)
- **Operator audit trail** JSON จากหน้า Technician (Copy JSON / Download JSON)
  หรือ `openthaiid audit` — รหัสผิดพลาดและ origin เท่านั้น ไม่มีข้อมูลบนบัตร
- ไฟล์ `openthaiid-qualify.json` ที่คำสั่งเขียน
- OS / เบราว์เซอร์ / เวอร์ชันตัวช่วย / รหัสเช่น `NO_CARD` `CARD_REMOVED`

ห้ามส่ง:

- ภาพบัตรหรือภาพฟอร์มที่กรอกแล้ว
- เลขประจำตัวประชาชน ชื่อ ที่อยู่ วันเกิด รูปถ่าย / `data:image`
- HAR หรือล็อกที่มี payload `card.read`

ถ้าไฟล์ไปมีเลขบัตรปน **ลบทิ้งแล้วรันใหม่** อย่ามาแดงด้วยมือในแชท
