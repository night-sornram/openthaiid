# วิธีใช้ OpenThaiID (เจ้าหน้าที่คลินิก)

ใช้ได้เฉพาะคอมพิวเตอร์ที่ต่อเครื่องอ่านบัตร ไม่มีเว็บไซต์ OpenThaiID
และไม่มีอะไรต้องติดตั้งบนคลาวด์

หน้าไฟสถานะเริ่มต้นเป็นภาษาไทย (**พร้อม**). ถ้าต้องการภาษาอังกฤษ
สลับเป็น **English** บนหน้า Ready หรือตั้ง `OPENTHAIID_LANG=en`
เครื่องมือช่างเทคนิคยังเป็นภาษาอังกฤษ

## ทุกวัน

1. **เปิด OpenThaiID บนเครื่องนี้** ถ้ายังไม่ทำงาน:
   - Windows: OpenThaiID ใน Start Menu หรือ **Start OpenThaiID** ในโฟลเดอร์ตัวช่วย
   - macOS: **OpenThaiID.app** (ไอคอนบนแถบเมนู — ยังไม่เซ็นชื่อ ดู
     [installation.md](installation.md))
   - Linux: เมนูแอป **OpenThaiID** หรือ `openthaiid tray`
2. **เสียบเครื่องอ่านบัตร**
3. **เปิดหน้าคลินิกตามปกติ**
4. สถานะควรขึ้น **พร้อม** (ตัวช่วยทำงาน เครื่องอ่านต่อแล้ว เสียบบัตรแล้ว)
   เมนู **แสดงสถานะ** ใช้ไฟสามดวงเดียวกัน และไม่แสดงข้อมูลบนบัตร
   บนหน้า พร้อม กด **ทดสอบการเชื่อมต่อ** ได้ (ไม่อ่านข้อมูลบนบัตร)
   ถ้าตัวช่วยหลุดระหว่างวัน ไม่ต้องปิดหน้านี้ — รอให้กลับมา แล้วกด กรอกข้อมูล
5. **เสียบบัตรประชาชนของผู้ป่วย**
6. กด **กรอกข้อมูล** (Fill / Fill from Thai ID)

ถ้าแผนก IT เปิด **เริ่มเมื่อเข้าสู่ระบบ** หรือติดตั้ง always-on
(`openthaiid service install`) ข้ามข้อ 1

ครั้งแรกที่หน้าคลินิกขออ่านบัตร OpenThaiID จะแสดง
**เว็บไซต์ต้องการอ่านบัตร** อนุญาตเฉพาะหน้าคลินิกที่กำลังใช้
ถ้ารหัสบนหน้าคลินิกมี ให้รหัสตรงกัน หลังจากนั้นกรอกข้อมูลได้จนกว่าจะปิด
OpenThaiID หรือสิทธิ์หมดอายุ

เมนูถาด: **เปิดหน้า พร้อม** · **Technician tools** · **แสดงสถานะ** ·
**เริ่มเมื่อเข้าสู่ระบบ** (เลือกเปิด) · **ออก**

## “พร้อม” หมายถึงอะไร

หน้าตัวช่วย (`http://127.0.0.1:17891/`) กล่องสถานะในหน้าคลินิก และ
**แสดงสถานะ** ใช้ไฟสามดวงเดียวกัน:

| ไฟ | ปกติเมื่อ |
|---|---|
| OpenThaiID บนเครื่องนี้ | ตัวช่วยกำลังทำงาน |
| เครื่องอ่านบัตร | เสียบเครื่องอ่านแล้ว |
| บัตรประชาชนไทย | เสียบบัตรแล้ว |

หน้าเหล่านี้ไม่แสดงเลขบัตร ชื่อ หรือข้อมูลส่วนบุคคลอื่น

## ถ้ายังไม่พร้อม

| ที่เห็น | ลองทำ |
|---|---|
| OpenThaiID ยังไม่ทำงาน | เปิดตัวช่วย (Start Menu / แอป / `openthaiid tray`) หน้านี้รอให้กลับมาได้เอง |
| ตัวช่วยหลุดการเชื่อมต่อ | ไม่ต้องปิดหน้าคลินิก — รอไฟสถานะ แล้วกด กรอกข้อมูล อีกครั้ง |
| ทดสอบการเชื่อมต่อ | บนหน้า พร้อม กด **ทดสอบการเชื่อมต่อ** (ไม่อ่านข้อมูลบนบัตร) |
| ไม่พบเครื่องอ่านบัตร | ถอดแล้วเสียบเครื่องอ่านใหม่ |
| เสียบบัตรประชาชนไทย | เสียบบัตรให้สุด |
| เว็บไซต์ต้องการอ่านบัตร | กดอนุญาตหน้าคลินิก (ให้รหัสตรงกัน) |
| ยังติด | เรียกช่างเทคนิค — **Technician tools** จากถาด หรือ `#technician` |

ห้ามส่งภาพหน้าจอของบัตรผู้ป่วย หรือของช่องที่กรอกแล้ว
หน้าตัวช่วยไม่ต้องการค่านั้นเพื่อหาสาเหตุ

## ฝึกใช้ก่อนได้เครื่องจริง

ยังไม่มี ACR39U ก็ฝึกได้: เปิด `openthaiid -synthetic` แล้วใช้ตัวอย่างคลินิก
ตาม [training.md](training.md) ครั้งแรกยังต้องกดอนุญาต
ห้ามส่งภาพหน้าจอหรือค่าในช่องไปในทิคเก็ต

วันศุกร์ที่มีเครื่องอ่าน: [friday-checklist.md](friday-checklist.md)

---

# Using OpenThaiID (clinic staff) — English

You only need the computer next to the card reader. There is no OpenThaiID
website to visit and nothing to deploy in the cloud.

Staff-facing Ready lights default to **Thai**. Use the Ready page
**English** toggle, or `OPENTHAIID_LANG=en` for tray / `openthaiid status`.
Technician tools stay English.

## Every day

1. **Start OpenThaiID on this computer** if it is not already running:
   - Windows: OpenThaiID in the Start Menu, or **Start OpenThaiID** in the
     helper folder (notification-area icon).
   - macOS: **OpenThaiID.app** (menu-bar helper; unsigned — see
     [installation.md](installation.md)).
   - Linux: application menu **OpenThaiID**, or `openthaiid tray`.
2. **Plug in the card reader.**
3. **Open the clinic ERP** as usual.
4. Status should say **พร้อม / Ready** (helper running, reader plugged in,
   card inserted). Tray **แสดงสถานะ** uses the same three lights and never
   shows personal card data. On the Ready page, **ทดสอบการเชื่อมต่อ / Test
   connection** runs a safe probe (no personal fields). If the helper
   restarts, leave the page open — it detects the helper coming back.
5. **Insert the patient’s Thai ID card.**
6. Click **Fill** (กรอกข้อมูล).

If clinic IT turned on **Start at login** or installed the always-on
service (`openthaiid service install`), skip step 1.

The first time that clinic page asks to read a card, OpenThaiID shows
**เว็บไซต์ต้องการอ่านบัตร / A website wants to read the card**. Allow only
the clinic page you are using. Match the code if the ERP shows one.

Do not send screenshots of the patient’s card or of filled form fields.

Practice without a reader: [training.md](training.md) (`openthaiid -synthetic`).
Friday with ACR39U: [friday-checklist.md](friday-checklist.md).

