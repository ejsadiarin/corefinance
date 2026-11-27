# corefinance

A personal budget service behind coregateway: users record what they spend and earn, set up recurring rules for the regular ones, and read their budget back as totals, trends and breakdowns.

## Language

### Ownership

**Owner**:
The user a record, tag or rule belongs to, known to corefinance only as the identity the gateway asserts. corefinance keeps no other user data: the owner's timezone is the gateway's (coregateway ADR-0005). corefinance remembers only the zone it last saw on a request, for work done without one (the worker).
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

**Occurred at**:
The moment a budget record happened, when the owner gives a time: their wall clock on the record's date, read in their timezone and kept as UTC. Optional; rule entries and records entered by day have none. The record's date stays what every figure counts, so a later timezone change moves how the time reads, never the day.
_Avoid_: timestamp, transaction time, created at

**Category**:
One slice of a fixed, system-defined list that a budget record is filed under, at most one per record. Expenses and incomes have separate lists, and owners can't add to either (ADR-0002).
_Avoid_: tag, type, bucket

**Reimbursement**:
Income that pays the owner back for something they spent, such as a friend's share of a dinner. It counts in money on hand but not as income in reports.
_Avoid_: refund, repayment, payback

**Tag**:
A free label an owner creates and puts on any number of expenses or incomes, across categories: people, trips, anything a category doesn't capture.
_Avoid_: category, label

**Priority**:
Whether a record is a need, a want or savings, the three groups the 50/30/20 analysis splits a budget into.
_Avoid_: importance, tier, class

**Skipped**:
A budget record the owner has marked as not happening. It is kept, but it no longer counts in the totals.
_Avoid_: deleted, cancelled, voided

**Period budget**:
The budget for the day, week or month containing a date, read in the owner's timezone. Weeks run Monday to Sunday. It is funded by the period's income, spends through today, and sets aside bills still due.
_Avoid_: monthly budget, allocation, spending limit

**Funded**:
The money a period budget has: posted income in the period, excluding reimbursements, plus projected income.
_Avoid_: budget total, allowance, income total

**Projected income**:
Income an active recurring rule will still write in the rest of the period. It uses the same due dates and exchange rates the materialize pass writes with, and skips dates an entry already exists for.
_Avoid_: expected income, forecast, estimate

**Safe to spend**:
What is left in a period budget divided by the days remaining in the period, counting today. For a future period every day is remaining; for a past period it is empty.
_Avoid_: daily budget, spend per day, allowance

### Money

**Currency**:
The ISO 4217 code an amount is entered in. It must be one the exchange-rate source publishes.
_Avoid_: denomination, unit

**Reporting currency**:
PHP, the one currency every report totals in.
_Avoid_: base currency, home currency

**Original amount**:
The amount of a budget record in the currency it was entered in.
_Avoid_: gross amount, foreign amount

**Exchange rate**:
The PHP value of one unit of a record's original currency on its rate date.
_Avoid_: conversion rate, FX rate

**Rate date**:
The date whose published exchange rate a record uses: its own date, or the previous business day with a published rate.
_Avoid_: effective date, conversion date

**PHP amount**:
The original amount converted at the exchange rate and rounded to centavos, fixed when the record is saved.
_Avoid_: converted amount, local amount

**Money on hand**:
The owner's money as of today: every income dated today or earlier, less every expense dated today or earlier, in the owner's timezone. Reimbursements count; skipped records don't.
_Avoid_: total money, balance, current balance

**Committed**:
Money already promised to future-dated expenses: the total of not-skipped expenses dated after today in the owner's timezone.
_Avoid_: pending, upcoming, reserved

**Average daily spend**:
Net spend over the trailing 30 days ending today — expenses less reimbursements, in the owner's timezone — divided by 30 calendar days.
_Avoid_: burn rate, daily burn, spending rate

**Runway**:
Money on hand divided by the average daily spend, in whole days. It is empty when there is no spending to divide by.
_Avoid_: days left, buffer

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

### Export

**Budget part**:
The owner's budget as one part of a coregateway export: tags, rules and budget records, with the state that stops a rule repeating itself (how far it has filled, whether it is paused, which entries were edited). Importing it keeps every id, or, for another account's file, gives every row an id derived for the owner the same way coreban does; either way a coreban note's link to a record survives, and what the owner already has is skipped.
_Avoid_: backup, dump, budget export file
