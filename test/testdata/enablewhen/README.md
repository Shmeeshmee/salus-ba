# enableWhen test vectors

Shared cases for the three evaluators of FHIR R4 `enableWhen` (docs/clinic-cms, package Q1): the server
(`internal/questionnaireresponse/service_enable.go`), the web builder's preview (`web/src/enableWhen.ts`) and the
patient app (`mobile/lib/enable_when.dart`). Each must pass every case here unchanged.

Each file is one questionnaire and its cases:

- `questionnaire.item`: the FHIR R4 `Questionnaire.item` tree, only the elements the evaluators read (`linkId`,
  `text`, `type`, `repeats`, `answerOption`, `item`, `enableWhen`, `enableBehavior`). Every questionnaire here is
  one the server publishes.
- `cases[].answers`: by `linkId`, the answers given, as `QuestionnaireResponse.item.answer` values (`valueBoolean`,
  `valueInteger`, `valueDecimal`, `valueString`, `valueDate`). Answers to items that end up hidden are given on
  purpose: they count as no answer.
- `cases[].enabled`: every enabled item's `linkId` (sections and instructions included), in document order. Every
  other item is hidden.

The rules the cases pin:

- An item is enabled when its parent is and its conditions hold: all of them (`enableBehavior` `all`, or a single
  condition) or at least one (`any`).
- A condition reads the answers of its question only while that question is enabled; a hidden question counts as
  unanswered (R4).
- `exists`: `answerBoolean` true holds when the question has an answer, false when it has none.
- Every other operator is false on an unanswered question, `!=` included.
- `=` holds when any answer equals the condition's answer; `!=` when none does (a multiple choice compares each
  answer). `>`, `>=`, `<`, `<=` compare numbers (integer, decimal) and dates (`YYYY-MM-DD`, earlier is less).
- A condition names only an earlier question, so one pass in document order decides every item.
