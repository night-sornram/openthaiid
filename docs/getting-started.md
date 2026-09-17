# เริ่มต้นใช้ OpenThaiID (เช้าวันศุกร์)

หนึ่งหน้าสำหรับเช้านี้ — เปิดอันนี้ก่อน แล้วตามลิงก์ด้านล่าง
ยังไม่มีเครื่องอ่าน ACR39U ก็ฝึกได้ด้วยโหมดฝึกซ้อม

**ห้าม** ส่งเลขบัตร ชื่อ ที่อยู่ วันเกิด รูปถ่าย ภาพหน้าจอฟอร์ม หรือ HAR
ที่มียอด `card.read`

ซอฟต์แวร์พร้อมที่ **v0.1.0-rc.7** — เช้าวันศุกร์คือ ACR39U จริง
([NOT_DONE.md](../NOT_DONE.md)). ไม่มี GitHub Actions ในรอบนี้ (นาทีหมดถึง
~1 ต.ค. 2026) และไม่มี Laser-id / NHSO

## เช้านี้เปิดอันไหน

| ใคร | เปิดเอกสารนี้ |
|---|---|
| เจ้าหน้าที่คลินิก (ทุกวัน — เปิดตัวช่วย เสียบเครื่องอ่าน กดกรอกข้อมูล) | [user.md](user.md) |
| เช้าวันศุกร์ — บัตรโต๊ะเจ้าหน้าที่ (พิมพ์หนึ่งหน้า) | [friday-desk-card.md](friday-desk-card.md) |
| ฝึกก่อนได้เครื่องจริง (`make friday-dry-run` / `openthaiid -synthetic`) | [training.md](training.md) |
| เช้าวันศุกร์มี ACR39U (พร้อม → qualify → กรอกข้อมูล → ส่งวินิจฉัย) | [friday-checklist.md](friday-checklist.md) |
| แผนก IT / ทีม ERP (ไลบรารีในหน้า Create Patient) | [erp-integration.md](erp-integration.md) |
| ช่างเทคนิคเมื่อกรอกข้อมูลไม่ขึ้น | [technician.md](technician.md) · HTTP บน loopback: [security.md](security.md) · HTTPS ของตัวช่วย: [loopback-tls.md](loopback-tls.md) · โต๊ะพร้อมหรือยัง: `openthaiid doctor` · ตรวจเวอร์ชัน: `openthaiid update check` |

หน้า พร้อม ของตัวช่วย: `http://127.0.0.1:17891/`
ไฟสามดวงมีคำว่า **ปกติ / รอ / ยังไม่พร้อม** ไม่ใช้สีอย่างเดียว
ปุ่ม **ใช้เครื่องนี้** · **ทดสอบการเชื่อมต่อ** · สลับภาษา ไทย / English กดง่ายขึ้น
Firefox / Safari ใช้หน้า พร้อม ได้; กรอกข้อมูลจากหน้าคลินิก **HTTPS** ใช้ได้เมื่อ
ช่างเทคนิคเปิด HTTPS ของตัวช่วย ([loopback-tls.md](loopback-tls.md)) —
มิฉะนั้น `HTTPS_WS_BLOCKED` ใช้ Chrome/Edge หรือหน้าคลินิก http
([compatibility.md](compatibility.md))

## ถ้ายังไม่มี ACR39U

1. **ซ้อมก่อนได้เครื่อง** — คำสั่งเดียว ซ้อมเส้นทางซอฟต์แวร์เช้าวันศุกร์บนบัตรจำลอง:

   ```bash
   make friday-dry-run
   # ซ้อมเลือกเครื่องอ่าน (สองเครื่องจำลอง):
   ./scripts/friday-dry-run.sh --two-readers
   # Windows: powershell -ExecutionPolicy Bypass -File scripts/friday-dry-run.ps1
   ```

   ไม่ต้องมี ACR39U และ**ไม่ใช่**ผ่านฮาร์ดแวร์ ถ้าผ่าน ให้เปิด [training.md](training.md)
2. เจ้าหน้าที่ฝึกกดกรอกข้อมูลตาม [user.md](user.md)
3. แผนก IT ผูกหน้า ERP ตาม [erp-integration.md](erp-integration.md)
4. เช้าวันศุกร์ที่มีเครื่องจริง พิมพ์ [friday-desk-card.md](friday-desk-card.md) ที่โต๊ะ
   แล้วใช้ชุด qualify [friday-checklist.md](friday-checklist.md)

## ถ้ามี ACR39U เช้านี้

1. พิมพ์ [friday-desk-card.md](friday-desk-card.md) หนึ่งหน้า วางที่โต๊ะเจ้าหน้าที่
2. เสียบเครื่องอ่าน เปิด OpenThaiID (**ไม่ใช้** `-synthetic`)
3. หน้า พร้อม ขึ้น **พร้อม** หลังเสียบบัตร — ถ้ามีหลายเครื่อง กด **ใช้เครื่องนี้**
4. กด **ทดสอบการเชื่อมต่อ** (ไม่อ่านข้อมูลบนบัตร)
5. ตาม [friday-checklist.md](friday-checklist.md) — qualify แล้วกรอกข้อมูล
6. ถ้ากรอกไม่ขึ้น เปิด [technician.md](technician.md) ส่ง Copy diagnostics / audit JSON เท่านั้น

ติดตั้งแพ็กเกจคลินิก: [installation.md](installation.md) (`START-HERE.txt`)
ชุดทดสอบฮาร์ดแวร์เต็ม: [hardware-qualification.md](hardware-qualification.md)
เว็บคลินิกปิดตัวช่วยไม่ได้ — ตาราง HTTP บน `127.0.0.1`: [security.md](security.md)
ช่างเทคนิคดูว่าโต๊ะพร้อมหรือยังด้วย `openthaiid doctor` (ไม่พิมพ์ข้อมูลบนบัตร)
แล้วตรวจว่ามี GitHub Release ใหม่หรือยัง (ไม่ดาวน์โหลดอัตโนมัติ):
หน้า พร้อม กด **ตรวจเวอร์ชัน** หรือ `openthaiid update check`
ไม่มีเน็ตก็กรอกข้อมูลได้ตามปกติ

---

# Getting started (Friday morning) — English

One index for this morning. Open this page first, then the linked guide.

Do **not** send citizen ID, names, address, date of birth, photos, filled-form
screenshots, or HARs that contain `card.read` payloads.

Software is **v0.1.0-rc.7**. Friday is physical ACR39U
([NOT_DONE.md](../NOT_DONE.md)). No GitHub Actions in this round (minutes
until ~1 Oct 2026). No Laser-id / NHSO.

| Who | Open |
|---|---|
| Clinic staff (every day) | [user.md](user.md) |
| Friday staff desk card (print one Thai page) | [friday-desk-card.md](friday-desk-card.md) |
| Practice before the reader arrives | [training.md](training.md) (`make friday-dry-run`) |
| Friday with an ACR39U | [friday-checklist.md](friday-checklist.md) |
| Clinic IT / ERP | [erp-integration.md](erp-integration.md) |
| Technician when Fill fails | [technician.md](technician.md) · loopback HTTP: [security.md](security.md) · helper TLS: [loopback-tls.md](loopback-tls.md) · desk ready?: `openthaiid doctor` · version check: `openthaiid update check` |

Ready page: `http://127.0.0.1:17891/`. The three lights include **OK /
Waiting / Not ready** text (not color alone). **Use this reader**, **Test
connection**, and the ไทย / English toggle are large tap targets.
Firefox/Safari: Ready works; Fill from an **https://** ERP needs optional
helper TLS ([loopback-tls.md](loopback-tls.md)) or returns `HTTPS_WS_BLOCKED`.
Use Chrome/Edge, an http ERP, or a trusted local cert. Do not disable
browser security.

Without hardware: `make friday-dry-run` then [training.md](training.md)
(ซ้อมก่อนได้เครื่อง). With ACR39U: print
[friday-desk-card.md](friday-desk-card.md) for the staff desk, then follow
[friday-checklist.md](friday-checklist.md). Package install:
[installation.md](installation.md). A clinic website cannot stop the
helper — loopback HTTP inventory: [security.md](security.md). Technicians can
check whether the desk is ready with `openthaiid doctor` (no personal fields),
then look for a newer GitHub Release (no download) with Ready **ตรวจเวอร์ชัน**
or `openthaiid update check`; offline does not block Fill.
