# Using OpenThaiID (clinic staff)

You only need the computer next to the card reader. There is no OpenThaiID
website to visit and nothing to deploy in the cloud.

## Every day

1. **Start OpenThaiID on this computer** (once, or leave it running).
2. **Plug in the card reader.**
3. **Open the clinic ERP** as usual.
4. Status should say **Ready** (helper running, reader plugged in, card
   inserted).
5. **Insert the patient’s Thai ID card.**
6. Click **Fill** (or **Fill from Thai ID**).

The first time that clinic page asks to read a card, OpenThaiID shows
**A website wants to read the card**. Allow only the clinic page you are
using. Match the code if the ERP shows one. After that, Fill works until
OpenThaiID is stopped or the grant expires.

## What “Ready” means

The helper page (`http://127.0.0.1:17891/`) and the status box on the ERP
use the same three lights:

| Light | OK means |
|---|---|
| OpenThaiID on this computer | The helper is running |
| Card reader | The reader is plugged in |
| Thai ID card | A card is inserted |

None of these screens show citizen ID, names, or other personal card data.

## If it is not Ready

| You see | Try this |
|---|---|
| OpenThaiID is not running | Start the helper on this computer |
| No card reader | Unplug and plug the reader back in |
| Insert the Thai ID card | Seat the card fully |
| A website wants to read the card | Allow the clinic page (match the code) |
| Still stuck | Ask a technician — they have a hidden **Technician / developer** section |

Do not send screenshots of the patient’s card or of filled form fields.
The helper page never needs those values to diagnose a problem.
