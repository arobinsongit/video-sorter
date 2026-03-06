---
name: smart-project
description: Create a new project with sequential numbering, folder structure, and metadata tracking.
---

# smart-project - Create New Project with Structure

**Dependencies:** Requires `git` to be installed.

**Purpose:** Create a new project with proper numbering, folder structure, and metadata tracking.
**Complexity:** Medium  
**Dependencies:** git, projects folder structure

---

## Overview

Creates a new project with sequential numbering (APS####), proper naming convention, standardized folder structure, and metadata tracking. Projects are created on a dedicated branch and merged after setup.

**Workflow:**
1. Determine next project number (APS####)
2. Collect project details (customer, system type, country, type)
3. Create project branch
4. Generate project structure
5. Create initial commit
6. Return to main branch (ready for smart-merge)

---

## Invocation

**Trigger phrases:**
- "smart project"
- "create project"
- "new project"
- "start project"
- "setup project"

---

## Project Numbering System

**Format:** `APS####` (e.g., APS0001, APS0002)

**Tracking:**
- Primary: `.project-registry.json` at repository root
- Validation: Check existing `projects/` folder for highest number
- Auto-increment from highest found
- Starts at APS0001 if no projects exist

---

## Project Naming Convention

**Format:** `APS####-CustomerName-SystemType-Country`

**Rules:**
- Remove all spaces from components
- Preserve original case (e.g., AcmeCorp, USA, PumpSystem)
- Separate components with hyphens
- No special characters except hyphens

**Examples:**
- External: `APS0001-AcmeCorp-PumpSystem-USA`
- External: `APS0002-TechIndustries-HVACControl-Canada`
- Internal: `APS0003-INTERNAL-InfrastructureUpgrade-UK`

---

## Project Types

### External Projects
- **Customer Name:** Actual customer/client name
- **Purpose:** Client-facing work, deliverables, documentation
- **Tracking:** Full metadata with customer details

### Internal Projects
- **Customer Name:** "INTERNAL" (fixed value)
- **Purpose:** Internal R&D, infrastructure, tools, planning
- **Tracking:** Same structure as external, customer = "INTERNAL"

---

## Interactive Creation Flow

### 1. Check Current State
```bash
# Ensure working directory is clean
git status

# Verify on main branch or offer to switch
git branch --show-current
```

### 2. Determine Next Project Number

**Step 2a: Read Registry**
```bash
# Check if .project-registry.json exists
# If not, create with: { "lastProjectNumber": 0, "projects": [] }
```

**Step 2b: Validate Against Existing Projects**
```bash
# List projects/ folder
# Parse APS#### from folder names
# Use highest of: registry number OR folder number
# Increment by 1
```

**Example:**
```
Registry shows: APS0002
Folders show: APS0001, APS0002, APS0003
Next number: APS0004 (highest found + 1)
```

### 3. Collect Project Details

**Ask user (in order):**

**Basic Information:**
```
1. Project Type:
   - External (customer project)
   - Internal (internal work)

2. Customer Name: (or "INTERNAL" if internal type)
   Example: "Acme Corp" → "AcmeCorp"

3. System Type:
   Example: "Pump System" → "PumpSystem"

4. Country:
   Example: "USA", "UK", "Canada"

5. Project Summary: (brief description)
```

**Primary Contact:** (skip if Internal project)
```
6. Contact Name:
7. Contact Email:
8. Contact Phone:
9. Contact Role/Title:
```

**Addresses:** (skip if Internal project)
```
10. Customer Billing Address: (full address)
11. Is delivery address same as billing? (yes/no)
    → If no: Delivery Address:
12. Is site installation address same as billing? (yes/no)
    → If no: Site Installation Address:
```

**Commercial:** (skip if Internal project)
```
13. Quote Reference Number:
14. Purchase Order (PO) Number:
15. Sales Order (SO) Number:
16. Contract Value:
17. Currency: (e.g., GBP, EUR, USD)
18. Exchange Rate to USD: (e.g., 1.27 for GBP→USD)
    → Auto-calculate USD equivalent
19. Payment Terms: (e.g., "Net 30", "50% upfront, 50% on delivery")
```

**Project Team:**
```
20. Sales Representative:
21. Project Sponsor:
```

**Timeline:**
```
22. Start Date: [AUTO-SET to today's date - 2026-01-20]
23. Estimated Project Length: (in business weeks, e.g., 12)
    → Auto-calculate end date (start + weeks, round to following Friday)
```

**Validate inputs:**
- Remove spaces from customer name, system type
- Preserve case
- Format dates consistently (YYYY-MM-DD)
- Calculate USD value: contract value × exchange rate
- Calculate end date: add business weeks, find next Friday
- Confirm all details with user before proceeding

### 4. Generate Project Identifier

**Format:** `APS####-CustomerName-SystemType-Country`

**Example:**
```
Number: APS0004
Customer: Acme Corp → AcmeCorp
System: Pump System → PumpSystem
Country: USA

Result: APS0004-AcmeCorp-PumpSystem-USA
```

### 5. Check for Similar Projects

**Scan existing projects:**
```bash
# Find all .project-meta files in projects/ folder
# Parse JSON metadata from each file
# Compare against new project data
```

**Similarity Detection Logic:**

🔴 **CRITICAL - Exact Duplicate:**
- Same customer (case-insensitive)
- Same system type (case-insensitive)  
- Status: Active
- **Action:** Warn strongly, show existing project details, require explicit confirmation

🟡 **WARNING - Similar Project:**
- Same customer (case-insensitive)
- Different system type
- Status: Active
- **Action:** Notify about active project with customer, ask to confirm

🟢 **INFO - Previous Project:**
- Same customer (case-insensitive)
- Any system type
- Status: Completed or Closed
- **Action:** Show previous project(s) for reference, auto-continue unless user stops

**Example Warning:**
```
🔴 CRITICAL: Potential duplicate project detected!

Existing Project:
  Number: APS0002
  Identifier: APS0002-AcmeCorp-PumpSystem-UK
  Customer: Acme Corp
  System: Pump System
  Country: UK
  Status: Active
  Created: 2025-11-15

New Project:
  Number: APS0004
  Identifier: APS0004-AcmeCorp-PumpSystem-USA
  Customer: Acme Corp
  System: Pump System
  Country: USA

This appears to be a duplicate or very similar project.
Are you sure you want to create this? (yes/no/view)
  'yes' - Create anyway
  'no' - Cancel
  'view' - Open existing project
```

**No Matches:**
```
✓ No similar projects found
✓ Ready to create APS0004-AcmeCorp-PumpSystem-USA
```

### 6. Create Project Branch

```bash
# Create and checkout project branch
git checkout -b project/APS0004-AcmeCorp-PumpSystem-USA

# Verify branch created
git branch --show-current
```

### 7. Create Project Structure

**Folder structure:**
```
projects/APS0004-AcmeCorp-PumpSystem-USA/
├── README.md                     # Project overview
├── .project-meta                 # Machine-readable metadata (JSON)
├── admin/                        # Pre-project commercial/legal documents
│   ├── README.md
│   ├── nda/
│   ├── quotations/
│   ├── purchase-orders/
│   ├── sales-orders/
│   ├── forecasts/
│   └── customer-urs/
├── project-management/           # Active project execution tracking
│   ├── README.md
│   ├── meetings/
│   ├── status-reports/
│   ├── decisions/
│   ├── planning/
│   ├── risks-issues/
│   └── communications/
├── docs/                         # Technical documentation
│   └── README.md
├── notes/                        # Development notes, research
│   └── README.md
├── data/                         # Project data files
│   └── README.md
├── analysis/                     # Analysis, calculations
│   └── README.md
└── deliverables/                 # Final deliverables
    └── README.md
```

**Note:** For Internal projects, admin subfolders (nda, quotations, purchase-orders, sales-orders) are marked as "NOT REQUIRED" but still created for consistency.

### 8. Generate README.md

**Template:**
```markdown
# Project APS0004: AcmeCorp Pump System

**Project Number:** APS0004  
**Customer:** Acme Corp  
**System Type:** Pump System  
**Country:** USA  
**Type:** External  
**Status:** Active  
**Created:** 2026-01-20  
**Start Date:** 2026-01-20  
**Estimated End Date:** 2026-04-18 _(12 weeks)_  

---

## Project Summary

[User-provided summary]

---

## Contact Information

**Primary Contact:**  
- Name: John Smith  
- Email: john.smith@acmecorp.com  
- Phone: +1-555-0123  
- Role: Engineering Manager  

**Addresses:**  
- **Billing:** 123 Main St, New York, NY 10001, USA  
- **Delivery:** Same as billing  
- **Site Installation:** 456 Factory Rd, Buffalo, NY 14201, USA  

---

## Commercial

**Purchase Order:** PO-2026-0123  
**Sales Order:** SO-2026-0456  
**Quote Reference:** Q-2026-0789  
**Contract Value:** £50,000 GBP (USD $63,500)  
**Payment Terms:** 50% upfront, 50% on delivery  

---

## Project Team

**CKS Scientific:**  
- **Sales Representative:** Jane Doe  
- **Project Sponsor:** Bob Johnson  

**Customer:**  
- **Primary Contact:** John Smith (Engineering Manager)

---

## Project Structure

- **admin/** - Pre-project commercial and legal documents
  - nda/ - Non-disclosure agreements
  - quotations/ - Customer quotations
  - purchase-orders/ - Customer purchase orders
  - sales-orders/ - Sales order acknowledgments
  - forecasts/ - Project forecasts
  - customer-urs/ - Customer user requirement specifications
- **project-management/** - Active project execution and tracking
  - meetings/ - Meeting notes and minutes
  - status-reports/ - Regular status updates
  - decisions/ - Decision logs and records
  - planning/ - Project plans and schedules
  - risks-issues/ - Risk register and issue tracking
  - communications/ - Client/stakeholder correspondence
- **docs/** - Technical documentation
- **notes/** - Research and development notes
- **data/** - Project data files
- **analysis/** - Analysis and calculations
- **deliverables/** - Final deliverables for customer

---

## Quick Links

- [Admin](admin/README.md)
- [Project Management](project-management/README.md)
- [Documentation](docs/README.md)
- [Notes](notes/README.md)

---

## Status Log

| Date | Status | Notes |
|------|--------|-------|
| 2026-01-20 | Created | Project initialized |

---

**Last Updated:** 2026-01-20
```

### 9. Generate .project-meta File

**Pure JSON metadata file:**

**Template:**
```json
{
  "number": "APS0004",
  "identifier": "APS0004-AcmeCorp-PumpSystem-USA",
  "customer": "Acme Corp",
  "systemType": "Pump System",
  "country": "USA",
  "type": "External",
  "status": "Active",
  "created": "2026-01-20",
  "startDate": "2026-01-20",
  "estimatedEndDate": "2026-04-18",
  "estimatedWeeks": 12,
  "lastUpdated": "2026-01-20",
  "phase": "Initialization",
  "completionPercent": 0,
  "contact": {
    "name": "John Smith",
    "email": "john.smith@acmecorp.com",
    "phone": "+1-555-0123",
    "role": "Engineering Manager"
  },
  "addresses": {
    "billing": "123 Main St, New York, NY 10001, USA",
    "delivery": "Same as billing",
    "siteInstallation": "456 Factory Rd, Buffalo, NY 14201, USA"
  },
  "commercial": {
    "quoteRef": "Q-2026-0789",
    "purchaseOrder": "PO-2026-0123",
    "salesOrder": "SO-2026-0456",
    "contractValue": 50000,
    "currency": "GBP",
    "usdValue": 63500,
    "exchangeRate": 1.27,
    "paymentTerms": "50% upfront, 50% on delivery"
  },
  "team": {
    "salesRep": "Jane Doe",
    "projectSponsor": "Bob Johnson"
  },
  "tags": [
    "customer:AcmeCorp",
    "system:PumpSystem",
    "country:USA",
    "type:external",
    "status:active"
  ],
  "history": [
    {
      "date": "2026-01-20",
      "event": "Project Created",
      "notes": "Initialized project structure",
      "user": "[Git user]",
      "branch": "project/APS0004-AcmeCorp-PumpSystem-USA"
    }
  ]
}
```

### 10. Update Project Registry

**Update .project-registry.json:**

Add entry to the registry file at repository root:

```json
{
  "lastProjectNumber": 4,
  "projects": [
    {
      "number": "APS0001",
      "identifier": "APS0001-PreviousCustomer-System-Country",
      "status": "Closed",
      "created": "2025-12-01"
    },
    {
      "number": "APS0004",
      "identifier": "APS0004-AcmeCorp-PumpSystem-USA",
      "customer": "Acme Corp",
      "systemType": "Pump System",
      "country": "USA",
      "type": "External",
      "status": "Active",
      "created": "2026-01-20",
      "branch": "project/APS0004-AcmeCorp-PumpSystem-USA",
      "lastUpdated": "2026-01-20"
    }
  ]
}
```

### 11. Create Subfolder READMEs

**admin/README.md:**
```markdown
# Project Administration

Pre-project commercial and legal documents.

## Structure

- **nda/** - Non-disclosure agreements
- **quotations/** - Customer quotations and pricing
- **purchase-orders/** - Customer purchase orders
- **sales-orders/** - Sales order acknowledgments
- **forecasts/** - Project forecasts and budgets
- **customer-urs/** - Customer user requirement specifications

## Document Lifecycle

These documents are typically created before project execution begins:
1. NDA signed
2. Quotations prepared and sent
3. Purchase order received
4. Sales order acknowledgment sent
5. Project forecast created
6. Customer URS collected

## Internal Projects

For Internal projects (Customer = "INTERNAL"):
- ⚠️ nda/ - NOT REQUIRED (no external customer)
- ⚠️ quotations/ - NOT REQUIRED (no external quotes)
- ⚠️ purchase-orders/ - NOT REQUIRED (no customer PO)
- ⚠️ sales-orders/ - NOT REQUIRED (no sales process)
- ✅ forecasts/ - REQUIRED (internal planning)
- ✅ customer-urs/ - REQUIRED (internal stakeholder requirements)

## Sensitive Information

⚠️ This folder may contain sensitive commercial information:
- Pricing details
- Contract terms
- Financial forecasts

Be mindful when committing to git. Consider using links to secure storage (e.g., Google Drive) for highly sensitive documents.
```

**project-management/README.md:**
```markdown
# Project Management

Active project execution and tracking documents.

## Structure

- **meetings/** - Meeting notes and minutes
- **status-reports/** - Regular status updates
- **decisions/** - Decision logs and records
- **planning/** - Project plans and schedules
- **risks-issues/** - Risk register and issue tracking
- **communications/** - Client/stakeholder correspondence

## Usage

This folder is actively updated throughout project lifecycle:

### During Project Execution
- Log meeting notes after each meeting
- Update status reports regularly (weekly/monthly)
- Document key decisions as they're made
- Track risks and issues as they arise
- Keep project plan current
- Archive important communications

### Document Naming

Use consistent naming conventions:
- Dates: `YYYY-MM-DD` format
- Versions: `v1`, `v2`, `v3`
- Examples:
  - `2026-01-15_meeting-notes.md`
  - `2026-01_status-report.md`
  - `decision-001_system-architecture.md`

### Best Practices

- Date all documents
- Link to related documents
- Keep notes concise and actionable
- Update status regularly
- Document decisions with rationale
- Track action items from meetings

## vs Admin Folder

**admin/** = Pre-project documents (NDA through Sales Order)  
**project-management/** = Active execution documents (meetings, tracking, decisions)
```

### 12. Create Additional Subfolder READMEs

**docs/README.md:**
```markdown
# Project Documentation

Store all project documentation here:

- Requirements
- Specifications
- Design documents
- User guides
- Technical documentation

## Organization

Organize by document type or project phase as appropriate.
```

**notes/README.md:**
```markdown
# Project Notes

Development notes, research, and working documents:

- Research notes
- Meeting notes
- Decision logs
- Ideas and brainstorming
- Working documents

## Tips

- Date your notes
- Use descriptive filenames
- Link to related documents
```

**data/README.md:**
```markdown
# Project Data

Store project data files here:

- Input data
- Configuration files
- Data exports
- Reference data

## Data Management

- Document data sources
- Note data formats
- Track data versions if needed
```

**analysis/README.md:**
```markdown
# Analysis & Calculations

Analysis work and calculations:

- Spreadsheets
- Calculation documents
- Analysis reports
- Models and simulations

## Organization

Organize by analysis type or date as appropriate.
```

**deliverables/README.md:**
```markdown
# Project Deliverables

Final deliverables for customer/stakeholders:

- Reports
- Presentations
- Final documentation
- Code/software releases
- Data packages

## Deliverable Tracking

Track deliverable status and delivery dates in main project README.
```

### 13. Final Commit

**Update .project-registry.json:**
```json
{
  "lastProjectNumber": 4,
  "projects": [
    {
      "number": "APS0001",
      "identifier": "APS0001-PreviousCustomer-System-Country",
      "status": "Closed",
      "created": "2025-12-01"
    },
    {
      "number": "APS0004",
      "identifier": "APS0004-AcmeCorp-PumpSystem-USA",
      "customer": "Acme Corp",
      "systemType": "Pump System",
      "country": "USA",
      "type": "External",
      "status": "Active",
      "created": "2026-01-20",
      "branch": "project/APS0004-AcmeCorp-PumpSystem-USA",
      "lastUpdated": "2026-01-20"
    }
  ]
}
```

### 13. Create Initial Commit

```bash
# Stage all project files
git add projects/APS0004-AcmeCorp-PumpSystem-USA/
git add .project-registry.json

# Create conventional commit
git commit -m "feat(project): initialize APS0004-AcmeCorp-PumpSystem-USA

- Project Number: APS0004
- Customer: Acme Corp
- System Type: Pump System
- Country: USA
- Type: External
- Created full project structure with docs, notes, data, analysis, deliverables
- Updated project registry"
```

### 14. Push and Prepare for Merge

```bash
# Push project branch
git push -u origin project/APS0004-AcmeCorp-PumpSystem-USA

# Return to main
git checkout main

# Inform user
echo "✅ Project APS0004 created on branch: project/APS0004-AcmeCorp-PumpSystem-USA"
echo "Next steps:"
echo "  1. Review project structure"
echo "  2. Use 'smart-merge' to merge project into main"
echo "  3. Use 'smart-cleanup' after merge"
```

---

## Integration with Other Skills

### After Project Creation

**Use smart-merge:**
```bash
smart-merge
# Merges project branch into main
# Strategy: merge (--no-ff) to preserve project initialization
```

**Use smart-cleanup:**
```bash
smart-cleanup
# Removes merged project branch
```

### During Project Work

**Create feature branches:**
```bash
smart-branch
# Example: feat/APS0004-add-specifications
# Include project number in branch name for tracking
```

**Save progress:**
```bash
smart-save
# Quick checkpoints during project work
```

**Organize commits:**
```bash
smart-commit
# Organize project changes into logical commits
```

### With smart-status

**smart-status enhancements:**
- Show active projects (status: Active)
- Show project branches
- Quick navigation to project folders
- Display project count and recent activity

---

## Project Lifecycle

### 1. Active Projects
- Status: "Active"
- Currently being worked on
- Regular updates and commits

### 2. On Hold Projects
- Status: "OnHold"
- Temporarily paused
- Document reason in .project-meta

### 3. Completed Projects
- Status: "Completed"
- All deliverables finished
- Archive or keep for reference

### 4. Closed Projects
- Status: "Closed"
- No longer active
- Can be archived to separate folder

---

## Error Handling

### Registry Conflicts
```
Error: Registry shows APS0005 but folders show APS0007
Action: Use APS0008 (highest + 1)
Warning: Registry may be out of sync
```

### Duplicate Project Identifier
```
Error: Project APS0004-AcmeCorp-PumpSystem-USA already exists
Action: Offer to:
  1. View existing project
  2. Create with different parameters
  3. Cancel
```

### Invalid Characters
```
Error: Customer name contains invalid characters: "Acme & Corp"
Suggestion: Remove/replace: "AcmeAndCorp" or "AcmeCorp"
```

### Missing Projects Folder
```
Action: Create projects/ folder automatically
Inform: "Creating projects/ folder for first project"
```

---

## Configuration

**Optional .smartproject.json:**
```json
{
  "projectPrefix": "APS",
  "numberLength": 4,
  "numberStart": 1,
  "defaultFolders": [
    "docs",
    "notes",
    "data",
    "analysis",
    "deliverables"
  ],
  "projectTypes": ["External", "Internal"],
  "statusValues": ["Active", "OnHold", "Completed", "Closed"],
  "registryFile": ".project-registry.json",
  "projectsFolder": "projects",
  "branchPrefix": "project/",
  "autoMerge": false,
  "teamMembers": []
}
```

---

## Example Session

```
User: smart-project
Claude: Checking current state...
 On branch: main
 Working directory: clean

Determining next project number...
 Registry shows: APS0003
 Projects folder shows: APS0001, APS0002, APS0003
 Next project number: APS0004

Creating new project APS0004...

1. Project Type:
   1) External (customer project)
   2) Internal (internal work)

User: 1

Claude: 2. Customer Name (will remove spaces):

User: Acme Corp

Claude: 3. System Type (will remove spaces):

User: Pump System

Claude: 4. Country:

User: USA

Claude: 5. Project Description (one line):

User: Industrial pump control system with IoT monitoring

Claude: 
 Project Identifier: APS0004-AcmeCorp-PumpSystem-USA

Confirm creation? (yes/no)

User: yes

Claude:
Creating project structure...
 Created branch: project/APS0004-AcmeCorp-PumpSystem-USA
 Created folder: projects/APS0004-AcmeCorp-PumpSystem-USA/
 Created README.md
 Created .project-meta
 Created admin/ structure (6 subfolders)
 Created project-management/ structure (6 subfolders)
 Created docs/ folder
 Created notes/ folder
 Created data/ folder
 Created analysis/ folder
 Created deliverables/ folder
 Updated registry: .project-registry.json

Committing project...
 Committed: feat(project): initialize APS0004-AcmeCorp-PumpSystem-USA

Pushing to remote...
 Pushed to origin/project/APS0004-AcmeCorp-PumpSystem-USA

Returning to main branch...
 Switched to main

 Project APS0004 created successfully!

 Location: projects/APS0004-AcmeCorp-PumpSystem-USA/
 Branch: project/APS0004-AcmeCorp-PumpSystem-USA

Next steps:
1. Review project structure
2. Use 'smart-merge' to merge project into main
3. Use 'smart-cleanup' after merge
4. Start working: cd projects/APS0004-AcmeCorp-PumpSystem-USA/
