# Record Classification

Record classification is a method of determining where the record is at, and
what it's class is. Classificiation is used mainly for determining where the
record should be filed but can also apply to other elements of gramophile,
like validation tasks etc.

## Background

In RecordCollection we explicitly classified records as one of a proto enum value,
but in gramophile we'd prefer to keep it more freeform and allow users to classify
records however they wish. Thus we are not going to use explicit classification
but instead use inferred classification - i.e. there are inherent properties
of a record which lead it into a given class and then we use that derived class
to make determinations about where the record should be filled.

## Example

So for example, if a record does not have a cleaning date, it should go into the
cleaning pile. If the record has a cleaning date, and we're trying to move it into
a listening pile, and the cleaning date is over e.g. 2 years old, we should also
move it into the listening pile. We could classifiy a record as "needs_clean" as
follows:

```
cleaning_date < 0 -> needs_clean
cleaning_date > "2 years ago" -> needs_clean

needs_clean -> EVALUATE_ON_MOVE
```

Thus we have two rules that define needs_clean. This is not exhaustive since > 0 < "2 years ago" does not mean needs_clean applies.

## Rule overlap

To deal with rule overlap - i.e. the possibility that two rules can apply to a given record, and how to
decide between them. We deal with this by applying strict precedence to our rules, and then take whatever
the first rule that actively applies to the record. To help users with fixing their categories we highlight
both the chosen category and the rule that led us to it when we call a gram get.

## Implementation

1. Define empty classification rule proto
1. Refine classification rule proto - start with allowing bools
1. Expand classification to support ints
1. Expand classification to support int64 dates
1. Implement classification logic
1. Return logic and rule on a gram get
