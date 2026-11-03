---
title: "Anastasya"
---

import ActionsTable from "@site/src/components/Actions/ActionsTable";
import AoETable from "@site/src/components/AoE/AoETable";
import FieldsTable from "@site/src/components/Fields/FieldsTable";
import FramesTable from "@site/src/components/Frames/FramesTable";
import HitlagTable from "@site/src/components/Hitlag/HitlagTable";
import IssuesTable from "@site/src/components/Issues/IssuesTable";
import NamesList from "@site/src/components/Names/NamesList";
import ParamsTable from "@site/src/components/Params/ParamsTable";

## Known issues

<IssuesTable item_key="anastasya" />

Anastasya is implemented from a pre-release leak. Base stats are copied from Ganyu, all frames and particle counts are placeholders, and talent tables are extrapolated from a single talent level.

- `anastasya skill` (tap) triggers the next stage of Azure Star Decree - Annihilation while an unsealed party member is waiting, instead of switching to that character.
- Azure Star Decree - Triumph is triggered by swapping to Anastasya after her own Sealing ends.
- Sealed characters cannot be swapped to; the swap waits until the Sealing ends.
- Sealing reduction (1s per Stellar Glimmer DMG instance with a 1s ICD for Benevolence, 1s per Normal Attack for Glory) and the 10s unsealing windows are assumptions.
- A4: the Polestar Field bonus is modelled as +1 stack per buff refresh, Stellar Prism creation is not implemented, and "Level 2" Stellar Vortexes start at 3 stacks.

## Names

<NamesList item_key="anastasya" />

## Frames

<FramesTable item_key="anastasya" />

## Hitlag Data

<HitlagTable item_key="anastasya" />

## AoE Data

<AoETable item_key="anastasya" />

## Legal Actions

<ActionsTable item_key="anastasya" />

## Params

<ParamsTable item_key="anastasya" />

## Fields

<FieldsTable item_key="anastasya" />
