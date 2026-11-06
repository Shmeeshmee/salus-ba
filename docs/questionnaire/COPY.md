# Questionnaires: text for review

Status: approved by the owner, 2026-11-06, with the owner's changes (W11, W35, M13); W41 and W42 approved as
written the same day. Packages 7 to 10 put these strings into `notification`, `web/src/messages.ts` and
`mobile/lib/messages.dart` exactly as approved here. A change to any of them goes back to the owner first. W is
the web app (practitioners and admins), M the mobile app (patients), E the e-mails. [Brackets] are filled in
by the app.

## Navigation and library (web)
- W1: "Questionnaires" (navigation link and page title)
- W2: "New questionnaire"
- W3 (status filter): "All", "Drafts", "Active", "Retired"
- W4 (status): "Draft", "Active", "Retired"
- W5 (tooltip on the status): "Version [n], last changed [date]"
- W6 (empty library): "No questionnaires yet. Create one to ask patients questions before their appointments."
- W7 (empty filter): "No questionnaires with this status."
- W8 (title of a new draft until changed): "Untitled questionnaire"

## Builder (web)
- W9: "Title", "Description (optional)"
- W10: "Add an item", with the hint "Drag an item into the questionnaire, or select it to add it at the end."
- W11 (item types): "Section", "Instructions", "Short answer", "Long answer", "Yes or no", "Single choice", "Multiple choice", "Whole number", "Decimal number", "Date"
- W12 (empty questionnaire): "Drag items here, or select one on the left."
- W13 (item fields): "Question", "Instructions", "Section title", "Required"
- W14 (under the title): "This questionnaire is a template for all patients. Do not include any patient's details."
- W15 (choice options): "Options", "Add option", "Option [n]", "Remove option"
- W16: "More options", "Add help text", "Help text", with the hint "Patients see this when they tap the information icon."
- W17 (condition): "Add condition", "Show this only if", "is", "Remove condition"; when there is nothing to depend on: "A condition needs a yes or no question or a choice question above this item."
- W18 (item actions): "Move up", "Move down", "Delete", "Drag to reorder"
- W19 (deleting a question others depend on): "Delete this question?" / "[n] items are shown only after an answer to it. Their conditions will be removed." with the buttons "Delete", "Cancel"
- W20: "Save draft", "Saving…", "Saved", "Discard draft"
- W21: "Discard this draft?" / "This cannot be undone." with the buttons "Discard", "Cancel"
- W22: "Publish"; "Publish this questionnaire?" / "Once published, it cannot be changed. To change it later, you create a new version." For a later version, also: "Version [n] will be retired. Patients who have already been asked keep that version." Buttons "Publish", "Cancel"
- W23 (active version): "This questionnaire is published and cannot be changed." with the buttons "Edit as new version", "Retire"
- W24: "Retire this questionnaire?" / "It is no longer sent with new bookings or added to appointments, including as a default questionnaire. Patients who have already been asked can still submit it." with the buttons "Retire", "Cancel"
- W25 (retired version): "This questionnaire is retired. It is no longer sent to patients."
- W26 (leaving with unsaved changes): "You have unsaved changes. Leave without saving?" with the buttons "Leave", "Stay"
- W27 (limits): "[n] of 200 items"; "A questionnaire can have up to 200 items."; "[n] of [max] characters" (shown near the limit); "A choice question needs at least 2 options."; "A choice question can have up to 50 options."; "Each option must be different."; "Sections cannot be placed inside other sections."
- W28 (item errors, on the item): "Add text to this item."; "This condition depends on a question that is no longer above this item."; "This condition's answer is no longer one of the options."
- W29 (errors): `draft_exists`: "A draft of this questionnaire already exists." with the button "Open draft"; 412: "Someone else changed this questionnaire. Reload to see their changes. Your changes since the last save will be lost." with the button "Reload"; `not_draft`: "This version is published and can no longer be changed."; 422: "Some items need attention."; not found: "This questionnaire is not available."

## Drag and drop for screen readers (web)

Needed because `@dnd-kit` (D7) speaks to screen readers during a drag and its own text is not ours. [Item] is
the item's text, or its type (W11) while it has none; [Section] the section's title.
- W41 (instructions, read when an item's drag handle gets focus): "To move this item, press space. Use the arrow keys to move it, space to drop it, or escape to cancel."
- W42 (announcements): "Moving [Item]."; "[Item] is at position [n] of [total]."; "[Item] is in [Section], at position [n] of [total]."; "[Item] moved to position [n] of [total]."; "Move cancelled. [Item] is back at position [n] of [total]." The last three are also announced after "Move up" and "Move down".

## Default questionnaire (web, Working hours)
- W30: "Questionnaire for every booking", with the choice "None" and the hint "Patients who book from now on are asked to fill it in before their appointment."
- W31 (`not_active`): "This questionnaire is not active. Choose an active one."

## Appointment detail (web)
- W32: "Questionnaires"
- W33 (none): "No questionnaires requested."
- W34 (status): "Not submitted yet", "Submitted [date and time]", "Withdrawn", "Not submitted before the appointment"
- W35: "Add questionnaire", "Questionnaire", "Add", "Cancel", with the hint "The patient sees it in the Salus app and is asked to submit it before the appointment starts."
- W36: "Withdraw"; "Withdraw this questionnaire?" / "The patient no longer needs to fill it in, and any answers they have saved are deleted." with the buttons "Withdraw", "Cancel"
- W37: "View answers", "Hide answers"
- W38 (answers): "Yes", "No", "Not answered"
- W39 (admin, tooltip on the status): "Only the appointment's practitioner can see the answers."
- W40 (errors): `already_requested`: "This questionnaire has already been requested for this appointment."; `not_active`: "This questionnaire is no longer active."; `appointment_started`: "The appointment has started, so questionnaires can no longer be added."; `consent_required`: "The patient's consent for this clinic is not active, so a questionnaire cannot be requested."; `submitted` (withdraw): "The patient has already submitted this questionnaire."

## My appointments (mobile)
- M1 (on the appointment's card, per questionnaire): "[Questionnaire title]" with its status "To fill in", "In progress", "Submitted", "Closed"
- M2: "Fill in", "Continue", "View answers"
- M3: "Please submit it before your appointment starts."
- M4 (closed): "The appointment has started, so this questionnaire can no longer be filled in."

## Filling in (mobile)
- M5 (at the top): "Your answers are saved as you go. [Clinic name] sees them only after you submit."
- M6 (saved state): "Saved", "Saving…", "Not saved. Check your connection. Saving will try again."
- M7: "Required"
- M8 (help icon): "More information"
- M9 (hints): "Your answer", "Choose a date", "Select one.", "Select all that apply.", "Yes", "No"
- M10 (field errors): "This question is required."; "Enter a whole number."; "Enter a number."; "This number is too large."; "Enter up to [max] characters."; "Enter a valid date."
- M11: "Submit"; "Submit your answers?" / "You cannot change them after you submit. Your practitioner at [Clinic name] will see them." with the buttons "Submit", "Cancel"
- M12 (submit with a missing answer): "Some required questions are not answered."
- M13 (submitted): "Answers submitted" / "Your practitioner can see them before your appointment. You can view them from My appointments."
- M14 (read-only view): "Your answers", "Submitted [date and time]", "Not answered"
- M15 (leaving while a save is pending): "Your last answer is not saved yet. Leave anyway?" with the buttons "Leave", "Stay"
- M16 (errors): `consent_required`: "Filling in this questionnaire needs your consent for this clinic."; `past_due`: "The appointment has started, so this questionnaire can no longer be changed."; `withdrawn`: "The clinic withdrew this questionnaire. You do not need to fill it in."; `submitted`: "You have already submitted this questionnaire."; 412: "These answers were changed on another device. The latest answers are shown."; not found: "This questionnaire is not available."

## E-mail
- E1 (added after the first paragraph of the booking confirmation and the 24-hour reminder, when a questionnaire is outstanding): "There is a questionnaire to fill in before your appointment. You can find it in the Salus app."

Never in an e-mail: the questionnaire's title, a question or an answer.
