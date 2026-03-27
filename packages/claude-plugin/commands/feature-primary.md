<!-- Efficiency: SDK calls: 2, Bash calls: 0, Context: ~4% -->

# /htmlgraph:feature-primary

Set the primary feature for activity attribution

## Usage

```
/htmlgraph:feature-primary <feature-id>
```

## Parameters

- `feature-id` (required): The feature ID to set as primary


## Examples

```bash
/htmlgraph:feature-primary feature-001
```
Set feature-001 as the primary feature for activity attribution



## Instructions for Claude

### Implementation:

**DO THIS:**

1. **Start the feature to set it as active (primary):**
   ```bash
   htmlgraph feature start <feature-id>
   ```

2. **Get feature details:**
   ```bash
   htmlgraph feature show <feature-id>
   ```

3. **List other active features:**
   ```bash
   htmlgraph find features --status in-progress
   ```

4. **Present a summary** using the output template below with:
   - feature_id: The ID of the primary feature
   - title: The feature title
   - other_active_features: List of other in-progress features

5. **Inform the user:**
   - All new activity will be attributed to this feature by default
   - Other features remain in progress and can be worked on

### Output Format:

## Primary Feature Set

**ID:** {feature_id}
**Title:** {title}

All subsequent activity will be attributed to this feature unless it matches another feature's patterns better.

### Other Active Features
{other_active_features}
