/**
 * Clinic ERP drop-in settings for "Create new patient".
 *
 * Copy this object (or build it from clinic admin settings) and pass it to
 * `autofillThaiID(config)` from a button click. Do not call it on page load.
 *
 * - `request`: protocol fields to read (subset of supported textual fields
 *   plus optional `portrait`). Nested pieces (`thaiName.givenName`) belong
 *   in `map`, not here. Default clinic request does **not** include portrait.
 * - `map`: slot → CSS selector on this screen. Phone / HN stay unmapped.
 * - Format keys are optional; they are already supported by autofillThaiID.
 * - Attach `onApprovalRequired` / `onError` in the click handler for ERP chrome.
 */

export const CLINIC_PATIENT_REQUEST = [
  "citizenId",
  "thaiName",
  "birthDate",
  "gender",
  "address",
];

/** Extra textual fields a clinic may opt into. Portrait is a separate opt-in. */
export const OPTIONAL_PATIENT_REQUEST = ["englishName"];
export const OPTIONAL_PORTRAIT_REQUEST = ["portrait"];

export const CLINIC_PATIENT_MAP = {
  citizenId: "#patient-cid",
  "thaiName.givenName": "#patient-given",
  "thaiName.familyName": "#patient-family",
  englishName: "#patient-english",
  birthDate: "#patient-dob",
  gender: "#patient-gender",
  address: "#patient-address",
  /** Example UI <img> only. Not requested unless portrait is in `request`. */
  portrait: "#patient-photo",
};

export const CLINIC_GENDER_LABELS = {
  male: "male",
  female: "female",
};

/** Build the object passed to `autofillThaiID` on Create Patient click. */
export function clinicPatientAutofill(overrides = {}) {
  const request = overrides.request ?? CLINIC_PATIENT_REQUEST;
  return {
    request,
    map: CLINIC_PATIENT_MAP,
    dateFormat: "iso",
    genderLabels: CLINIC_GENDER_LABELS,
    discover: false,
    ...overrides,
    request,
    map: overrides.map ?? CLINIC_PATIENT_MAP,
  };
}
