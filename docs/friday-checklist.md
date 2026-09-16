# วันศุกร์ — ACR39U (หนึ่งหน้า)

เสียบเครื่องอ่าน → พร้อม → qualify → กรอกข้อมูล → ส่งให้ jarvis เฉพาะวินิจฉัย
**ห้าม** ส่งเลขบัตร ชื่อ ที่อยู่ วันเกิด รูปถ่าย ภาพหน้าจอฟอร์ม หรือ HAR ที่มียอด `card.read`

ฝึกซ้อมโดยไม่มีเครื่อง: [training.md](training.md) (`-synthetic`)
ชุดเต็ม: [hardware-qualification.md](hardware-qualification.md)

## 1. เสียบเครื่องอ่าน → พร้อม

- เสียบ **ACR39U** เข้าเครื่องนี้ เปิด OpenThaiID (**ไม่ใช้** `-synthetic`)
  ถ้าแผนก IT ติดตั้ง always-on แล้ว (`openthaiid service install` /
  `Install-OpenThaiID-Service.ps1`) ไม่ต้องเปิดถาด — ดู
  [installation.md](installation.md)
- `openthaiid version` และ `openthaiid doctor` — `pcsc: OK` + ชื่อเครื่องอ่าน
  บรรทัด `personal fields: not read`
- หน้า Ready `http://127.0.0.1:17891/` ขึ้น **พร้อม** หลังเสียบบัตรประชาชน
- ถ้าเสียบเครื่องอ่านมากกว่าหนึ่งเครื่อง ให้กด **ใช้เครื่องนี้** บนหน้า พร้อม
  (หรือ Technician tools **Use this reader**) ก่อนกรอกข้อมูล
- กด **ทดสอบการเชื่อมต่อ** บนหน้า พร้อม (ไม่อ่านข้อมูลบนบัตร)
- ถ้า **ยังไม่พร้อม**: เปิดตัวช่วย → เสียบเครื่องอ่านใหม่ → เสียบบัตรให้สุด
  อย่าส่งภาพบัตร
- ถ้าตัวช่วยหลุดระหว่างวัน: เปิดใหม่ หน้า พร้อม / หน้าคลินิกรอให้กลับมาได้เอง
  แล้วกด กรอกข้อมูล อีกครั้ง (ไม่อ่านบัตรซ้ำเอง)

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
`BRIDGE_UNAVAILABLE` `NO_READER` `NO_CARD` `APPROVAL_DENIED` `FIELD_NOT_ALLOWED`)
แล้วเปิด Technician tools สำหรับ **Operator audit trail** (รหัสเท่านั้น)
ดู [technician.md](technician.md)

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
