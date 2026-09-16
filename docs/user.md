# Using OpenThaiID (clinic staff)

You only need the computer next to the card reader. There is no OpenThaiID
website to visit and nothing to deploy in the cloud.

## Every day

1. **Start OpenThaiID on this computer** if it is not already running:
   - Windows: OpenThaiID in the Start Menu, or **Start OpenThaiID** in the
     helper folder (notification-area icon).
   - macOS: **OpenThaiID.app** (menu-bar helper; unsigned — see
     [installation.md](installation.md)).
   - Linux: application menu **OpenThaiID**, or `openthaiid tray`.
2. **Plug in the card reader.**
3. **Open the clinic ERP** as usual.
4. Status should say **Ready** (helper running, reader plugged in, card
   inserted). Tray **Show status** uses the same three lights and never shows
   personal card data.
5. **Insert the patient’s Thai ID card.**
6. Click **Fill** (or **Fill from Thai ID**).

If clinic IT turned on **Start at login**, skip step 1 after you sign in.

The first time that clinic page asks to read a card, OpenThaiID shows
**A website wants to read the card**. Allow only the clinic page you are
using. Match the code if the ERP shows one. After that, Fill works until
OpenThaiID is stopped or the grant expires.

Tray / menu actions: **Open Ready page**, **Open Technician tools**,
**Show status**, **Start at login** (opt-in), **Quit**.

## What “Ready” means

The helper page (`http://127.0.0.1:17891/`), the status box on the ERP, and
the tray **Show status** dialog use the same three lights:

| Light | OK means |
|---|---|
| OpenThaiID on this computer | The helper is running |
| Card reader | The reader is plugged in |
| Thai ID card | A card is inserted |

None of these screens show citizen ID, names, or other personal card data.

## If it is not Ready

| You see | Try this |
|---|---|
| OpenThaiID is not running | Start the helper (Start Menu / app / `openthaiid tray`) |
| No card reader | Unplug and plug the reader back in |
| Insert the Thai ID card | Seat the card fully |
| A website wants to read the card | Allow the clinic page (match the code) |
| Still stuck | Ask a technician — **Open Technician tools** from the tray, or `#technician` |

Do not send screenshots of the patient’s card or of filled form fields.
The helper page never needs those values to diagnose a problem.
