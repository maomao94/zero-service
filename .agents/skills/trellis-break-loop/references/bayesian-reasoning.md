# Thinking Framework: Bayesian Reasoning

When multiple root causes are plausible and evidence is incomplete, update your beliefs proportionally to new evidence rather than clinging to initial assumptions.

## Step 1: Establish Priors

Before investigating, state what you believe and why:

| Hypothesis | Prior | Reasoning |
|------------|-------|-----------|
| H1: [cause A] | 40% | Most common for this pattern |
| H2: [cause B] | 30% | Plausible given environment |
| H3: [other] | 30% | Catch-all |

Priors must sum to 100%. If you can't assign probabilities, investigate first.

## Step 2: Observe Evidence

Document what you found — be specific about reliability:

- What exactly did you observe?
- How reliable? (test output > log message > user report > hunch)
- Could multiple hypotheses explain this?

## Step 3: Update Beliefs

For each hypothesis, ask: **How likely is this evidence if this hypothesis were true?**

Direction of update matters more than calculation:
- Evidence strongly predicted by H1 → H1 probability increases
- Evidence contradicts H2 → H2 probability decreases
- Evidence equally likely under all → no update

## Step 4: Seek Discriminating Evidence

Don't gather more of the same. Find evidence that **differs strongly** between top hypotheses.

> If H1 and H3 are close: "What would I see if H1 is true but not if H3 is true?" Then check for that.

## Step 5: State Confidence

| Confidence | Action |
|------------|--------|
| 90%+ | Proceed with fix, monitor |
| 70-90% | Proceed, add fallback check |
| 50-70% | Test hypothesis before committing |
| <50% | Need more evidence, don't guess |

Never express binary certainty when evidence is incomplete. Use "most likely", "plausible but unlikely", "worth investigating".

## Common Fallacies

| Fallacy | Example | Correction |
|---------|---------|------------|
| **Base rate neglect** | "Test failed → code is broken" | How often do tests fail for other reasons? |
| **Confirmation bias** | "Must be a race condition, let me find race evidence" | Actively seek evidence AGAINST your top hypothesis |
| **Anchoring** | "Last time it was caching, probably caching again" | Establish priors from current context, not yesterday's bug |
