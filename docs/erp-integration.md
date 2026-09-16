# เชื่อม ERP คลินิก / โรงพยาบาลที่มีอยู่แล้ว

ดัชนีเช้าวันนี้: [getting-started.md](getting-started.md)

OpenThaiID เป็นไลบรารีที่เรียกจากหน้า Create Patient (หรือ KYC) ของระบบที่มีอยู่แล้ว
**ไม่ต้องติดตั้งเว็บไซต์หรือแบ็กเอนด์ OpenThaiID**

ตัวช่วยบนเครื่องคลินิกยังต้องรันอยู่ (`openthaiid tray` หรือ `-synthetic` ตอนฝึก)
หน้า ERP อนุมัติตัวเองไม่ได้ — กด **อนุญาต** ที่หน้า พร้อม บนเครื่องนั้น

ห้ามเก็บเลขบัตร ชื่อ ที่อยู่ หรือรูปในไฟล์คอนฟิก และห้ามส่งค่าในช่องไปในทิคเก็ต

## ขั้นตอนแบบหยิบใส่

1. **เครื่องคลินิก** — ติดตั้งแพ็กเกจตาม [installation.md](installation.md) แล้วเปิด OpenThaiID
   (ฝึกใช้: `./dist/openthaiid -synthetic` — [training.md](training.md))
2. **โปรเจกต์ ERP** — ติดตั้ง `@openthaiid/web` (หรือ `@openthaiid/react`)
3. **คัดลอกคอนฟิก JSON** จากตัวอย่าง:
   - KYC สามช่อง: [`examples/clinic-patient/kyc-minimal.json`](../examples/clinic-patient/kyc-minimal.json)
     (`citizenId`, `thaiName`, `birthDate`)
   - ชุดคลินิก: [`examples/clinic-patient/clinic-full.json`](../examples/clinic-patient/clinic-full.json)
     (เพิ่ม `gender` + `address`; `englishName` / `portrait` แมปไว้แต่ไม่ขอจนกว่าจะใส่ใน `request`)
4. **แก้ `map`** ให้ชี้ selector ของฟอร์มใน ERP (`#hn` / เบอร์โทรอย่าแมป — ชิปบัตรไม่มี)
5. **ตรวจ JSON** ด้วยสคีมา
   [`sdk/web/schema/autofill-clinic-config.schema.json`](../sdk/web/schema/autofill-clinic-config.schema.json)
   (`parseAutofillClinicConfig` จาก `@openthaiid/web` ทำอย่างเดียวกันตอนรัน)
6. **กดปุ่มในหน้า** แล้วเรียกไลบรารี — คอลแบ็กอยู่ในโค้ด ไม่ใช่ใน JSON:

```js
import {
  autofillThaiID,
  formatStaffError,
  mountThaiIDStatus,
  parseAutofillClinicConfig,
  staffApprovalGuidance,
} from "@openthaiid/web";
import clinicConfig from "./openthaiid-clinic-config.json";

const config = parseAutofillClinicConfig(clinicConfig);

mountThaiIDStatus(document.getElementById("reader-status"));

fillBtn.addEventListener("click", async () => {
  await autofillThaiID({
    ...config,
    root: document.getElementById("patient-form"),
    onApprovalRequired: (info) => {
      const g = staffApprovalGuidance(info);
      showBanner(`${g.message} ${g.next}`);
    },
    onError: (err) => {
      showBanner(`${formatStaffError(err)} (${err.code ?? "ERROR"})`);
    },
  });
});
```

7. **ครั้งแรกของต้นทางนั้น** ให้พนักงานกดอนุญาตบนหน้า พร้อม (`http://127.0.0.1:17891/`)
8. **รูปถ่าย** เป็น opt-in — อย่าใส่ `portrait` ใน `request` จนกว่าหน้าจอต้องการ และห้าม `console.log` ไบต์รูป

Firefox / Safari กรอกข้อมูลจากหน้าคลินิก **HTTPS** ไม่ได้ (`HTTPS_WS_BLOCKED`)
ใช้ Chrome/Edge หรือหน้า http หน้า พร้อม ที่ `http://127.0.0.1:17891/` ใช้ได้
[compatibility.md](compatibility.md)

`request` เป็นรายการอนุญาตของฟิลด์โปรโตคอล (ไม่ใช่สล็อตซ้อนอย่าง `thaiName.givenName`)
`map` คือสล็อต → selector
`lang` เป็น `"th"` (ค่าเริ่มต้น) หรือ `"en"` — รหัสโปรโตคอลไม่เปลี่ยน
ถ้าถอดบัตรหรือถอดเครื่องอ่านระหว่างกรอก รหัสคือ `CARD_REMOVED` /
`READER_REMOVED` — `autofillThaiID` ล้างช่องที่แมปไว้ อย่าใช้ค่าค้าง

สิ่งที่**ห้าม**ใส่ใน JSON: `onApprovalRequired`, `onError`, `root`, `client`, `signal`,
ฟังก์ชัน `dateFormat`, และ Element ใน `map`

ตัวอย่างแผงตั้งค่า (ดาวน์โหลด/อัปโหลด JSON): เปิด
`examples/clinic-patient/` แล้วใช้ **config JSON** ด้านขวา

รายละเอียดห้องตรวจ: [clinic.md](clinic.md) · ช่างเทคนิค: [technician.md](technician.md)

---

# Drop-in ERP integration (English)

OpenThaiID is a JavaScript library you call from the hospital/clinic page you
already have. There is no OpenThaiID website or backend to deploy.

The PC with the reader still runs the local helper. The ERP cannot Allow
itself. Do not store personal field values in the config file.

## Steps

1. Install the local helper on clinic PCs ([installation.md](installation.md)).
   Practice with `-synthetic` until the ACR39U arrives ([training.md](training.md)).
2. Depend on `@openthaiid/web` (or `@openthaiid/react`).
3. Copy a sample config:
   - Minimal 3-field KYC: `examples/clinic-patient/kyc-minimal.json`
   - Fuller clinic set: `examples/clinic-patient/clinic-full.json`
4. Point `map` at your form selectors. Leave phone / HN unmapped.
5. Validate against
   `sdk/web/schema/autofill-clinic-config.schema.json`
   (package export `@openthaiid/web/schema/autofill-clinic-config.json`)
   or `parseAutofillClinicConfig()`.
6. Call `autofillThaiID({ ...config, root, onApprovalRequired, onError })`
   from a visible click. Callbacks are **code-only**.
7. Allow that origin on the Ready page the first time.
8. Keep `portrait` out of `request` unless the screen needs the photo.
   Do not log photo bytes.

Firefox/Safari Fill from an **https://** ERP is blocked (`HTTPS_WS_BLOCKED`).
Use Chrome/Edge or an http page. Ready at `http://127.0.0.1:17891/` still
works. [compatibility.md](compatibility.md).

TypeScript: `AutofillClinicConfig` in `@openthaiid/web`.

The clinic example settings panel can download/upload this JSON so integrators
can iterate without editing `settings.js`.
