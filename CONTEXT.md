# corefinance

A personal budget service behind coregateway: users record what they spend and earn, set up recurring rules for the regular ones, and read their budget back as totals, trends and breakdowns.

## Language

### Ownership

**Owner**:
The user a record, category or rule belongs to, known to corefinance only as the identity the gateway asserts. corefinance keeps no other user data: the owner's timezone is the gateway's (coregateway ADR-0005). corefinance remembers only the zone it last saw on a request, for work done without one (the worker).
_Avoid_: account, member, principal

**Last-seen zone**:
The timezone corefinance last saw on one of the owner's requests. The worker dates things in it, and an owner it has never seen a zone for is on UTC. It is a copy of the gateway's zone, never set any other way.
_Avoid_: user settings, preferred timezone, local timezone

### Budget

**Budget record**:
An expense or an income. It is what every total, trend and export counts, and what a coreban note links.
_Avoid_: transaction, entry, item

**Expense**:
Money the owner spent, on one date. Always a single record: a repeating one is a recurring rule's entries, never one expense with a schedule.
_Avoid_: cost, spending, payment, debit

**Income**:
Money the owner received, on one date. Like an expense, always a single record.
_Avoid_: earning, revenue, credit, deposit

**Category**:
An owner's named grouping for expenses or for incomes; the two kinds are kept apart. Removing one hides it rather than deleting the records filed under it.
_Avoid_: tag, type, bucket

**Tag**:
A free label an owner puts on any number of expenses or incomes, across categories.
_Avoid_: category, label

**Priority**:
Whether a record is a need, a want or savings, the three groups the 50/30/20 analysis splits a budget into.
_Avoid_: importance, tier, class

**Skipped**:
A budget record the owner has marked as not happening. It is kept, but it no longer counts in the totals.
_Avoid_: deleted, cancelled, voided

### Recurring rules

**Recurring rule**:
A schedule that produces expenses or incomes: an amount, a category and a cadence, from a start date, with an optional end date. Its entries are real budget records, written when they fall due in the owner's timezone (ADR-0001).
_Avoid_: recurring expense, recurring income, subscription, template

**Cadence**:
How often a recurring rule repeats: daily, weekly, monthly or yearly, counted from its start date. A monthly or yearly rule falls on the last day of a month too short to have its day (Jan 31, then Feb 28).
_Avoid_: frequency, interval, recurrence type

**Rule entry**:
An expense or income a recurring rule produced, for one of its due dates. Once written it is an ordinary budget record that the owner can edit, skip or delete; a deleted one is not written again.
_Avoid_: instance, occurrence, generated expense

**Edited entry**:
A rule entry the owner has changed or skipped since its rule wrote it. When the rule's schedule changes, an edited entry is kept and reported for review rather than skipped.
_Avoid_: modified instance, override, exception

**Paused**:
A recurring rule that writes no entries for now. Resuming it carries on from today without filling the paused stretch, and the entries it wrote before the pause stay.
_Avoid_: inactive, disabled, stopped

**Materialize pass**:
The one writer of rule entries: for one owner, it writes every rule's entries that have fallen due since the last pass, up to today. The worker runs it on a schedule, and saving a rule runs it for that rule's owner.
_Avoid_: backfill, generation, sync, cron

**Reconcile**:
What happens to a rule's existing entries when its schedule (start, end or cadence) changes: untouched entries the new schedule no longer produces are skipped, and edited ones are kept for review. Nothing is deleted, and dates before today are never filled.
_Avoid_: rebuild, regenerate, cleanup

**What's new**:
The feed of rule entries in the order they were written, so the owner can review what their rules posted since they last looked.
_Avoid_: inbox, activity log, notifications
