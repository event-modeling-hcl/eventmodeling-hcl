# Shared Event Model edit policy

This is the single edit policy for the
[business Event Modeling skill](../SKILL.md) and the
[Event Modeling HCL skill](../../event-modeling-hcl/SKILL.md).
Apply it before changing an existing model in any medium. Tools do not enforce
these permissions. Reading and reviewing a locked workflow remain allowed.

## Workflow permissions

A workflow's status controls permission to change it:

| Status | Without named approval | With approval for this workflow and these changes |
| --- | --- | --- |
| Absent or `created` | Editable within the user's requested scope. | Make the requested changes. |
| `planned`, `assigned`, `in_progress`, `review`, `blocked`, `informational` | Locked. Read and reference only. | Make only the named changes. |
| `done` | Locked. Read and reference only. | Make only the named changes and state that the edit reopens the work. |

A lock covers content and identity. Do not change, move, rename, or delete a
locked workflow without approval that names the workflow and the change.
This includes fields, examples, scenarios, edges, ownership, chapter membership,
and its position in the story. A file split, merge, or reorganization cannot
move a locked workflow without that approval.

An unclear "ok" is not approval. Approval for one workflow does not unlock its
siblings or every future change. A review or diagnostic does not grant approval.
An explicit user request that names the workflow and the intended changes
supplies approval for those changes only.

Do not set a status to its existing value. Do not advance statuses through the
work lifecycle unless the user asks. For completed work, do not change `done`
to another status unless the user explicitly approves reopening. Report the
reopening even when the approved edit does not change the recorded status.

## Connected changes

Before a rename, field change, or other change carried through a connected
chain, identify all affected workflows and their statuses. If any affected
workflow is locked, block the whole chain until the user approves its locked
parts. Do not leave a partial rename, mismatched field, duplicate contract,
compatibility alias, or broken reference in the unlocked part.

When a request also includes independent changes, make the permitted independent
changes. State exactly which chain or workflow stayed unchanged, its status,
and which approval is missing. Do not treat a blocked chain as permission to
replace the requested behavior with a smaller one.

## Shared contracts and consumers

Shared catalog entries are not an escape from workflow locks. Before changing
an event, aggregate, reusable field definition, identity, context, or other
shared contract, identify its consumers across the whole model. Include all
member files and indirect consumers whose meaning or references would change.

Compare the proposed contract with the fields, rules, examples, scenarios,
ownership, and references that each consumer relies on. A contract edit that
changes a locked consumer's meaning or requires its update is a change to that
locked workflow. Obtain named approval for that impact before changing the
shared contract or its affected chain.

Do not rename or move a shared definition to bypass this check. Do not copy the
same fact into another catalog entry or add an alias to avoid a locked consumer.
A compatible, unrelated addition can proceed only when it leaves locked
consumers and their contracts unchanged. If the impact is uncertain, report the
uncertainty and leave the shared contract unchanged until permission is clear.

## Questions and handoff

A hotspot records a question or dispute. It is not a workflow status or an
unlock. Its states are `open` and `resolved`, not the workflow lifecycle.
Resolve it only after recording the answer. Do not remove the concern without
a trace or use a status change to hide it.

Carry these permissions into any handoff between the two skills. Name the
locked workflows and consumers, the approved changes, any approved reopening,
and the connected changes that remain blocked. Both skills use this policy
directly rather than keeping separate copies of its lock rules.
