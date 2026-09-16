package qualify

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// OperatorStep is one E4–E7 / M1 prompt. Detail is English technician copy
// with a short Thai line. It never includes personal field values.
type OperatorStep struct {
	ID     string
	Name   string
	Prompt string
	Detail string
}

// OperatorChecks is the Friday operator slice (not automated).
func OperatorChecks() []OperatorStep {
	return []OperatorStep{
		{
			ID:     "E4",
			Name:   "removal during read / reader unplug",
			Detail: "pull card during Fill, then unplug reader; PASS if helper stays up and clinic error is CARD_REMOVED (card) / READER_REMOVED (unplug) / READ_FAILED. Staff Thai: reinsert, wait for พร้อม, Fill again.",
			Prompt: `E4 — Removal during read / reader unplug
ถอดบัตรขณะอ่าน แล้วถอดเครื่องอ่าน

Software maps Friday E4 to stable protocol codes (do not change the tokens):
  pull card during Fill/read  → CARD_REMOVED
  unplug USB reader mid-read  → READER_REMOVED
  native I/O that cannot tell  → READ_FAILED
Staff see Thai guidance (wait for พร้อม / Ready, then Fill again). Technician
English stays on this prompt, Copy diagnostics, and { lang: "en" }.

Do:
  1. Keep the helper running. Card seated. Do not type card data.
  2. Start Fill (or a qualify read). While it is reading, pull the card out.
     Clinic banner should say the card was removed; Ready must leave พร้อม.
  3. Re-seat the card, wait until Ready says พร้อม, Fill again.
  4. Start Fill, then unplug the USB reader.
     Clinic banner should say the reader was unplugged; Ready must leave พร้อม.

PASS if the helper process stays up and the error is CARD_REMOVED,
READER_REMOVED, or READ_FAILED — not a crash or hang.
FAIL if the helper dies, hangs, or Ready stays พร้อม with no card/reader.
SKIP if you cannot pull the card / unplug safely right now.

Never type citizen ID, names, address, dates of birth, or paste a photo.
ห้ามพิมพ์เลขบัตร ชื่อ ที่อยู่ หรือวางรูปถ่าย

Type pass, fail, or skip`,
		},
		{
			ID:     "E5",
			Name:   "browser refresh, bridge restart, sleep/wake",
			Detail: "Fill, refresh ERP, quit+start helper, sleep/wake. Map: helper down → BRIDGE_UNAVAILABLE + Test connection; after wake Ready re-probes (no card fields) and returns to พร้อม. PASS if reconnect works and logs have no field values.",
			Prompt: `E5 — Browser refresh, helper restart, sleep/wake
รีเฟรชเบราว์เซอร์ รีสตาร์ทตัวช่วย sleep/wake

Software maps Friday E5 to stable protocol codes and buttons (do not change the tokens):
  ERP / Ready cannot reach helper     → BRIDGE_UNAVAILABLE
  Ready / ทดสอบการเชื่อมต่อ while helper is down
                                      → Test connection fails (same code)
  After helper restart, leave Ready open
                                      → ตัวช่วยหลุดการเชื่อมต่อ then ตัวช่วยกลับมาแล้ว
  After OS sleep/wake                 → Ready re-probes (bridge.probe / status only,
                                        never card.read). If the helper is up but the
                                        card service is not → PCSC_UNAVAILABLE (see E6)

Do:
  1. Fill once from the clinic page (Allow if asked). Do not screenshot the form.
  2. Refresh the ERP tab. Fill again. Test connection on Ready must still be probe-only.
  3. openthaiid quit — Ready / clinic should show BRIDGE_UNAVAILABLE (Thai: ตัวช่วยยังไม่ทำงาน).
     Start the helper again. Leave the page open. Wait until พร้อม, Test connection, Fill again.
  4. Sleep the PC, wake it. Do not reboot. Confirm Ready re-probes by itself.
     If lights stay ยังไม่พร้อม, click ทดสอบการเชื่อมต่อ / Test connection.
     If the card service is down, use Technician tools → Reconnect card service, then wait for พร้อม.

PASS if reconnect works without a reboot, Fill works after each step, and technician logs
still have no citizen ID, names, address, or photo bytes.
FAIL if the helper will not come back, Fill hangs, or Ready stays พร้อม while the helper
or card service is down.
SKIP if you cannot sleep/wake this PC right now (record which step you did).

Never type citizen ID, names, address, dates of birth, or paste a photo.

Type pass, fail, or skip`,
		},
		{
			ID:     "E6",
			Name:   "missing/stopped PC/SC recovery",
			Detail: "stop pcscd / Smart Card service if safe; Ready leaves พร้อม; Test connection stays probe-only and shows PCSC_UNAVAILABLE; Reconnect card service; PASS when Ready says พร้อม again, or SKIP + reason",
			Prompt: `E6 — Missing/stopped PC/SC recovery
บริการอ่านบัตรหยุดทำงาน แล้วกลับมา

Software maps Friday E6 to stable protocol codes and buttons (do not change the tokens):
  Helper reachable, card service down → PCSC_UNAVAILABLE
  Ready / clinic ทดสอบการเชื่อมต่อ     → helper up, card service not ready (no personal fields)
  Technician Reconnect card service   → POST /v1/manager/reconnect
  After service is started again      → Ready must leave ยังไม่พร้อม and say พร้อม
                                        when the reader and card are back (no reboot)

Do (only if safe on this PC):
  Linux: stop pcscd briefly. Ready must leave พร้อม. doctor / Test connection / Fill
         show PCSC_UNAVAILABLE. Start pcscd. If lights stay down, click
         Reconnect card service. Wait until Ready says พร้อม. Fill again.
  Windows: stop the Smart Card service only if clinic IT allows it, then start it
         and Reconnect card service the same way.
  macOS: built-in PC/SC — SKIP if you cannot disable it; record SKIP. That is not a hardware fail.

PASS if you saw PCSC_UNAVAILABLE (or equivalent) and then recovery, helper still up,
Ready says พร้อม again, and no personal fields were logged.
FAIL if you needed a reboot, the helper crashed, or Ready stayed พร้อม while the
card service was down.
SKIP is expected on macOS when you cannot stop PC/SC.

Never dump card fields while the service is down.

Type pass, fail, or skip`,
		},
		{
			ID:     "E7",
			Name:   "local visual verification",
			Detail: "look at ERP / example on screen only; PASS if layout looks right AND nothing was copied or saved",
			Prompt: `E7 — Local visual verification (eyes only)
ตรวจบนหน้าจอเท่านั้น ห้ามบันทึก

Look at the clinic example / ERP preview. Confirm on screen only:
  - Thai / English name split
  - address parts
  - gender
  - photo appears (if portrait was requested)

PASS = it looked right AND you did not copy, screenshot, download, or paste
citizen ID, names, address, dates of birth, or photo bytes into a ticket.
FAIL = you needed to save field contents to judge (do not; stop and SKIP).

Type pass, fail, or skip`,
		},
		{
			ID:     "M1",
			Name:   "public HTTPS origin local-network permission",
			Detail: "Fill from a public https:// origin; record local-network allow/deny + helper grant; localhost HTTP is not this check",
			Prompt: `M1 — Public HTTPS local-network permission
หน้า https สาธารณะ (ไม่ใช่ localhost)

Localhost HTTP is NOT this check.

Do:
  1. Open the ERP (or examples/clinic-patient/) from a public https:// origin
     with a real certificate. Helper still on ws://127.0.0.1:17891/v1/ws.
  2. Click Fill. Allow local-network / local WebSocket if the browser asks.
  3. Allow the origin on the helper Ready page (match the comparison code).
     Keep portrait out of the grant unless the page opted in.

Record in the ticket (not in this prompt): browser + version, origin host
(no query string with personal data), local-network allow vs deny, helper
Allow vs deny.

PASS if Fill can proceed after those prompts (synthetic card is OK here).
SKIP if you have no public HTTPS origin today.

Type pass, fail, or skip`,
		},
	}
}

// ParseOperatorAnswer accepts only pass / fail / skip. Any other text —
// including anything that looks like personal or photo data — is refused.
func ParseOperatorAnswer(line string) (Result, bool) {
	raw := strings.TrimSpace(line)
	if raw == "" {
		return "", false
	}
	if ContainsForbidden(raw) || looksLikeCardData(raw) {
		return "", false
	}
	switch strings.ToLower(raw) {
	case "pass", "p", "yes", "y":
		return ResultPass, true
	case "fail", "f", "no", "n":
		return ResultFail, true
	case "skip", "s":
		return ResultSkip, true
	default:
		return "", false
	}
}

func looksLikeCardData(s string) bool {
	if thirteenDigitToken([]byte(s)) != "" {
		return true
	}
	// Operators must not paste payloads. Anything longer than a token is refused.
	if len([]rune(s)) > 16 {
		return true
	}
	if strings.Contains(strings.ToLower(s), "data:image") || strings.Contains(s, "/9j/") {
		return true
	}
	return false
}

// StdinOperator reads pass/fail/skip from a terminal. Non-TTY stdin is not used.
type StdinOperator struct {
	In    io.Reader
	Out   io.Writer
	IsTTY bool
	Log   func(string)
}

func (o StdinOperator) log(s string) {
	if o.Log != nil {
		o.Log(s)
	}
}

func (o StdinOperator) write(s string) {
	if o.Out == nil {
		return
	}
	if hit := FirstForbidden([]byte(s)); hit != "" {
		fmt.Fprintln(o.Out, "qualify: refused to print a prompt that looked like personal or photo data")
		return
	}
	fmt.Fprintln(o.Out, s)
}

// Ask prints a prompt and returns PASS/FAIL/SKIP. It never stores free text.
func (o StdinOperator) Ask(step OperatorStep) Result {
	o.write("")
	o.write(step.Prompt)
	if !o.IsTTY {
		o.write("stdin is not a terminal — recording SKIP. Re-run with -operator on a clinic PC keyboard.")
		return ResultSkip
	}
	r := bufio.NewReader(o.In)
	for attempt := 0; attempt < 5; attempt++ {
		o.write(">")
		line, err := r.ReadString('\n')
		if err != nil {
			o.log("operator input ended")
			return ResultSkip
		}
		if ContainsForbidden(line) || looksLikeCardData(strings.TrimSpace(line)) {
			o.write("Refused: that did not look like pass/fail/skip (do not type card data). Try again.")
			continue
		}
		res, ok := ParseOperatorAnswer(line)
		if ok {
			return res
		}
		o.write("Type pass, fail, or skip only.")
	}
	return ResultSkip
}

// Pause waits for Enter during hardware E3. It ignores/refuses typed card data.
func (o StdinOperator) Pause(prompt string) error {
	o.write(prompt)
	o.write("Press Enter when done. Do not type citizen ID, names, or paste a photo.")
	o.write("กด Enter เมื่อทำเสร็จ ห้ามพิมพ์ข้อมูลบนบัตร")
	if !o.IsTTY {
		return fmt.Errorf("stdin is not a terminal")
	}
	r := bufio.NewReader(o.In)
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return err
	}
	if ContainsForbidden(line) || thirteenDigitToken([]byte(line)) != "" {
		return fmt.Errorf("refused input that looked like personal or photo data")
	}
	return nil
}
