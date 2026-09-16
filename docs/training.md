# ฝึกใช้ก่อนได้เครื่องจริง

ใช้ตัวช่วยโหมดฝึกซ้อม (`-synthetic`) กับตัวอย่างคลินิก
**ไม่ต้องรอเครื่องอ่าน ACR39U** บัตรเป็นข้อมูลจำลอง ไม่ใช่ผู้ป่วยจริง

ครั้งแรกยังต้องกด **อนุญาต** บนหน้า พร้อม — หน้าคลินิกอนุมัติตัวเองไม่ได้

ห้ามส่งภาพหน้าจอ เลขบัตร ชื่อ ที่อยู่ หรือรูปถ่ายไปในทิคเก็ต
ส่งได้เฉพาะ Copy diagnostics, `openthaiid audit` JSON (รหัสเท่านั้น)
และรหัสผิดพลาด เช่น `NO_CARD`

## สั้น ๆ

1. สร้างตัวช่วยฝึกซ้อม:

   ```bash
   make build
   ./dist/openthaiid -synthetic
   ```

   หน้า Ready (`http://127.0.0.1:17891/`) ต้องขึ้น **พร้อม (บัตรฝึกซ้อม)**
   และมีกล่อง **ฝึกใช้ก่อนได้เครื่องจริง**

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

## เมื่อได้เครื่องจริง

ปิดตัวช่วยฝึกซ้อม แล้วเปิดใหม่ **โดยไม่ใส่** `-synthetic`
เสียบ ACR39U ตาม [friday-checklist.md](friday-checklist.md)
อย่าสับสนบัตรจำลองกับบัตรผู้ป่วย

รายละเอียดช่างเทคนิค: [technician.md](technician.md)

---

# Practice before the real reader arrives (English)

Start `./dist/openthaiid -synthetic`, open
`examples/clinic-patient/`, click Fill, then **Allow** on the Ready page.
The form shows fictional card data. Do not send screenshots or field
values in tickets — diagnostics, `openthaiid audit` JSON, and error
**codes** only.
Pass `{ lang: "en" }` / `?lang=en` for English staff banners.
When the ACR39U arrives, restart the helper without `-synthetic`.
