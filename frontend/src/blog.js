/**
 * MergeOS public blog catalog.
 * Posts are static for SSR/SEO reliability; Markdown mirrors live under /blog/*.md.
 */

export const blogPosts = [
  {
    slug: 'what-is-mergeos',
    title: 'What is MergeOS? The AI software delivery OS with escrow and proof',
    description:
      'MergeOS turns product briefs and GitHub repositories into funded tasks, human or AI delivery lanes, PR evidence, and public ledger proof for MRG rewards.',
    date: '2026-07-01',
    updated: '2026-07-12',
    author: 'MergeOS Team',
    tags: ['MergeOS', 'AI delivery', 'escrow', 'marketplace'],
    keywords: [
      'what is MergeOS',
      'AI software delivery OS',
      'developer bounty marketplace',
      'escrow software projects',
      'MRG token ledger',
    ],
  },
  {
    slug: 'escrow-backed-software-delivery',
    title: 'Escrow-backed software delivery: how MergeOS funds and releases work',
    description:
      'Learn how MergeOS reserves project budget in escrow, allocates task rewards, verifies PR and deployment evidence, and releases MRG with ledger hash-chain proof.',
    date: '2026-07-03',
    updated: '2026-07-12',
    author: 'MergeOS Team',
    tags: ['escrow', 'payments', 'payouts', 'proof'],
    keywords: [
      'software project escrow',
      'task bounty payouts',
      'escrow release workflow',
      'ledger proof',
      'MRG payouts',
    ],
  },
  {
    slug: 'ai-agents-and-human-bounties',
    title: 'AI agents and human bounties on one delivery graph',
    description:
      'Explore hybrid MergeOS workflows where AI agents scan repos, estimate rewards, review PRs, and humans claim funded bounties with shared evidence requirements.',
    date: '2026-07-05',
    updated: '2026-07-12',
    author: 'MergeOS Team',
    tags: ['AI agents', 'bounties', 'hybrid delivery'],
    keywords: [
      'AI coding agents',
      'human AI hybrid delivery',
      'developer bounties',
      'agent task routing',
      'PR review agents',
    ],
  },
  {
    slug: 'public-ledger-proof-explained',
    title: 'Public ledger proof explained: payouts, escrow events, and hash chains',
    description:
      'A practical guide to MergeOS public ledger logs: escrow reserves, task payments, manual credits, hash chaining, and how Scan explorer verifies accounts.',
    date: '2026-07-07',
    updated: '2026-07-12',
    author: 'MergeOS Team',
    tags: ['ledger', 'transparency', 'scan'],
    keywords: [
      'public ledger proof',
      'hash chain ledger',
      'escrow event logs',
      'MergeOS Scan',
      'payout transparency',
    ],
  },
  {
    slug: 'how-to-claim-mrg-bounties',
    title: 'How to claim MRG bounties on MergeOS (and sister repos like NeraJob)',
    description:
      'Step-by-step guide for contributors: star repos, claim issues, open evidence-backed PRs, pass review, and receive MRG credits on the MergeOS ledger.',
    date: '2026-07-09',
    updated: '2026-07-12',
    author: 'MergeOS Team',
    tags: ['contributors', 'MRG', 'bounties', 'how-to'],
    keywords: [
      'claim MRG bounty',
      'MergeOS contributor guide',
      'open source bounty workflow',
      'NeraJob bounties',
      'GitHub bounty claim',
    ],
  },
  {
    slug: 'nerajob-job-scanner-in-the-mergeos-ecosystem',
    title: 'NeraJob: global job scanning and CV tooling in the MergeOS ecosystem',
    description:
      'NeraJob is a Python toolkit for scanning public job boards, building CVs, and preparing applications—maintained under mergeos-bounties with MRG-funded scrapers.',
    date: '2026-07-12',
    updated: '2026-07-12',
    author: 'MergeOS Team',
    tags: ['NeraJob', 'jobs', 'Python', 'ecosystem'],
    keywords: [
      'NeraJob',
      'job board scraper',
      'CV builder Python',
      'MergeOS ecosystem',
      'remote jobs API',
    ],
  },
];

const bodies = {
  'what-is-mergeos': `# What is MergeOS?

**MergeOS** is an AI-assisted software delivery operating system. Customers fund product work, MergeOS turns briefs and repositories into claimable tasks, and contributors—humans, AI agents, or hybrid teams—deliver with **PR evidence** and **public ledger proof**.

## Why teams use MergeOS

Traditional freelancing and issue bounties often fail on three points: unclear scope, weak verification, and opaque payouts. MergeOS addresses each:

1. **Scoped tasks** from repository import, issue scoring, and AI workflow graphs  
2. **Escrow-backed budgets** with task reserves and platform fee accounting  
3. **Proof surfaces**: live feed, admin review, and hash-chained ledger entries  

## Core product surfaces

| Surface | What it does |
| --- | --- |
| Customer dashboard | Fund projects, monitor PRs, deployment, AI actions |
| Marketplace | Browse funded work and public bounties |
| Live feed | Realtime PR, payout, and agent events |
| Ledger / Scan | Public proof of mint, escrow, and payouts |
| Admin (uta) | Review, ops queue, manual MRG credits |
| SDK & protocol | Integrations for agents and external tools |

## Who it is for

- **Founders & SaaS teams** shipping features without expanding headcount permanently  
- **Repo owners** who want issue debt converted into funded, reviewable work  
- **Contributors** seeking transparent bounties with reputation and ledger receipts  
- **AI agents** that can claim structured task packets and submit evidence  

## Get started

- Product home: [https://mergeos.shop](https://mergeos.shop)  
- Marketplace: [https://mergeos.shop/marketplace](https://mergeos.shop/marketplace)  
- Whitepaper: [https://mergeos.shop/whitepaper](https://mergeos.shop/whitepaper)  
- GitHub: [mergeos-bounties/mergeos](https://github.com/mergeos-bounties/mergeos)  

Next: read [Escrow-backed software delivery](/blog/escrow-backed-software-delivery) for the money path.
`,

  'escrow-backed-software-delivery': `# Escrow-backed software delivery on MergeOS

When a customer funds a MergeOS project, budget is not left as a spreadsheet promise. The platform records **payment verification**, **platform fee**, **project reserve**, and **task-level payouts** so contributors can trust the reward path.

## High-level flow

1. Customer creates a project (brief or GitHub repo import)  
2. Payment is verified (PayPal, crypto rails, or configured providers)  
3. MergeOS mints internal **MRG** credit for the funded budget  
4. Platform fee is taken to treasury; remainder becomes work pool  
5. Tasks are reserved from the pool with acceptance criteria and evidence requirements  
6. After review/merge, task payment or manual credit hits the ledger  

## What “proof” means

Every meaningful money movement should leave a ledger row with:

- Type (\`project_reserve\`, \`task_payment\`, \`manual_credit\`, …)  
- From / to accounts (wallet, \`github:user\`, reserve, treasury)  
- Amount and reference (task id, PR URL)  
- Hash chaining for public verification on Scan  

## Why escrow matters for AI agents

Agents can execute quickly—but customers need control. Escrow + evidence requirements (tests, screenshots, security review) let MergeOS gate release without blocking parallel work.

## Practical tips for customers

- Fund enough budget that imported issues can receive non-trivial rewards  
- Prefer acceptance criteria that map to automated tests  
- Use project PR monitor and deployment validation before release  

Related: [Public ledger proof explained](/blog/public-ledger-proof-explained).
`,

  'ai-agents-and-human-bounties': `# AI agents and human bounties on one delivery graph

MergeOS is designed for **hybrid delivery**. The same project can route:

- **AI agent lanes** (scan, generate, review, test, deploy)  
- **Human contributors** claiming marketplace tasks  
- **Hybrid** work where agents prepare packets and humans finalize PRs  

## How routing works (conceptually)

1. Repository or brief intake produces tasks with complexity, reward, and suggested agent type  
2. Protocol manifests and SDK helpers expose task packets to external agents  
3. Humans see claimable work with acceptance and evidence checklists  
4. Admin or customer review gates payout  

## Evidence is the contract

Whether a worker is human or agent, MergeOS prefers explicit evidence:

- Tests and CI output  
- Screenshots for UI  
- Security review notes for sensitive paths  
- Deployment previews when required  

## Building agent integrations

Use the public protocol and SDK:

- Task and workflow schemas under \`/protocol\`  
- Live feed and WebSocket events  
- Agent action helpers in the JavaScript SDK  

If you are shipping a specialized tool in the ecosystem (for example job-market scrapers in **NeraJob**), fund implementation as MergeOS-linked bounties so delivery stays reviewable.

Related: [How to claim MRG bounties](/blog/how-to-claim-mrg-bounties).
`,

  'public-ledger-proof-explained': `# Public ledger proof explained

MergeOS publishes a **public ledger** so funding, reserves, and payouts are inspectable without exposing private customer secrets.

## What you can verify

- Project funding and token mint rows  
- Escrow / reserve movements  
- Task payments and manual credits to \`github:username\` or wallets  
- Hash chain integrity via verification APIs  

## Scan explorer

[scan.mergeos.shop](https://scan.mergeos.shop) presents ledger rows in a BscScan-style UI: addresses, sequences, and transaction-like references.

GitHub worker aliases such as \`github:alice\` and \`worker:github:alice\` are normalized so contributor payouts aggregate on one public identity where possible.

## For auditors and customers

1. Open the public ledger or Scan address page  
2. Confirm payout reference includes the merged PR URL when applicable  
3. Verify hash chaining with the public verify endpoint  

Transparency does not replace private security review—but it makes *economic* outcomes hard to rewrite silently.

Related: [Escrow-backed software delivery](/blog/escrow-backed-software-delivery).
`,

  'how-to-claim-mrg-bounties': `# How to claim MRG bounties

This guide is for contributors earning **MRG** on MergeOS and sister repositories (for example [NeraJob](https://github.com/mergeos-bounties/NeraJob)).

## 1. Star the repositories

Before bounty review:

- **Follow** [mergeos-bounties](https://github.com/mergeos-bounties)
- **Star** [mergeos-bounties/mergeos](https://github.com/mergeos-bounties/mergeos)
- **Star** [mergeos-bounties/mergeos-contracts](https://github.com/mergeos-bounties/mergeos-contracts)
- Star the product repo you will patch (e.g. NeraJob)  

## 2. Claim the issue

1. Comment on the bounty issue: \`I claim this bounty\`  
2. Comment on MergeOS [Claim Token issue #1](https://github.com/mergeos-bounties/mergeos/issues/1) with the issue URL  

## 3. Ship a focused PR

- Link the issue (\`Fixes #N\`)  
- Include tests and/or screenshots  
- Keep scope tight—no unrelated lockfile noise  
- Do not reintroduce free payment verifiers on production  

## 4. Pass maintainer review

Maintainers check:

- Star status  
- Evidence  
- CI  
- Security regressions  
- Merge conflicts  

## 5. Receive MRG credit

After merge, admin credits the ledger to \`github:<your-login>\` using the 25 / 50 / 100 / 200 scale. Issue titles may advertise higher marketing amounts; **final payout follows policy**.

## 6. Verify on Scan

Open [scan.mergeos.shop](https://scan.mergeos.shop) and search your \`github:username\` account for the manual credit or task payment row.

### Popular NeraJob work

Job board scrapers and CV tooling bounties live at:

[github.com/mergeos-bounties/NeraJob/issues](https://github.com/mergeos-bounties/NeraJob/issues?q=is%3Aissue+is%3Aopen+label%3Abounty)
`,

  'nerajob-job-scanner-in-the-mergeos-ecosystem': `# NeraJob in the MergeOS ecosystem

**NeraJob** is a Python toolkit for:

1. Scanning public job boards (pluggable scrapers)  
2. Building CVs from a local profile  
3. Preparing apply packages (cover notes + checklists)  

Repository: [mergeos-bounties/NeraJob](https://github.com/mergeos-bounties/NeraJob)

## Why it sits next to MergeOS

MergeOS funds and proves software delivery. NeraJob is a product surface in the same organization: contributors can claim **scraper and tooling bounties**, merge PRs, and receive **MRG** through the MergeOS ledger—the same economic rails as mergeos core work.

## Current capabilities (v0.1)

- CLI: \`nerajob profile\`, \`scan\`, \`cv\`, \`apply\`, \`jobs\`  
- Sample offline feed + RemoteOK adapter  
- Local JSON storage under \`data/\`  
- Extensible \`BaseScraper\` registry  

## Open bounty themes

- Remotive, Arbeitnow, Adzuna, USAJOBS, Greenhouse, Lever, and more  
- Rate-limited shared HTTP client  
- Match scoring and PDF CV export  
- Multi-source pack with offline CI mocks  

## Get started

\`\`\`bash
git clone https://github.com/mergeos-bounties/NeraJob.git
cd NeraJob
python -m venv .venv
# activate venv, then:
pip install -e ".[dev]"
nerajob profile init
nerajob scan -q "python" -n 20
\`\`\`

Claim workflow: [How to claim MRG bounties](/blog/how-to-claim-mrg-bounties) · [NeraJob BOUNTY.md](https://github.com/mergeos-bounties/NeraJob/blob/master/docs/BOUNTY.md)
`,
};

export function listBlogPosts() {
  return blogPosts
    .slice()
    .sort((a, b) => String(b.date).localeCompare(String(a.date)));
}

export function getBlogPost(slug = '') {
  const key = String(slug || '').trim().toLowerCase();
  const meta = blogPosts.find((post) => post.slug === key);
  if (!meta) return null;
  const body = bodies[key] || `# ${meta.title}\n\n${meta.description}\n`;
  return { ...meta, body, path: `/blog/${meta.slug}`, markdownPath: `/blog/${meta.slug}.md` };
}

export function blogSlugFromPath(path = '/') {
  const normalized = String(path || '/')
    .split('?')[0]
    .split('#')[0]
    .replace(/\/+$/, '') || '/';
  if (normalized === '/blog') return { page: 'blog', slug: '' };
  const match = normalized.match(/^\/blog\/([^/]+)$/);
  if (match) return { page: 'blog-post', slug: decodeURIComponent(match[1] || '') };
  return null;
}

/** Minimal Markdown → HTML for blog bodies (SSR-safe, no deps). */
export function markdownToHtml(markdown = '') {
  const lines = String(markdown || '').replace(/\r\n/g, '\n').split('\n');
  const html = [];
  let inUl = false;
  let inOl = false;
  let inCode = false;
  let codeBuffer = [];
  let paragraph = [];

  const flushParagraph = () => {
    if (!paragraph.length) return;
    const text = paragraph.join(' ').trim();
    paragraph = [];
    if (text) html.push(`<p>${inlineFormat(text)}</p>`);
  };

  const closeLists = () => {
    if (inUl) {
      html.push('</ul>');
      inUl = false;
    }
    if (inOl) {
      html.push('</ol>');
      inOl = false;
    }
  };

  for (const raw of lines) {
    const line = raw;
    if (line.startsWith('```')) {
      flushParagraph();
      closeLists();
      if (inCode) {
        html.push(`<pre><code>${escapeHtml(codeBuffer.join('\n'))}</code></pre>`);
        codeBuffer = [];
        inCode = false;
      } else {
        inCode = true;
      }
      continue;
    }
    if (inCode) {
      codeBuffer.push(line);
      continue;
    }
    if (!line.trim()) {
      flushParagraph();
      closeLists();
      continue;
    }
    const heading = line.match(/^(#{1,3})\s+(.*)$/);
    if (heading) {
      flushParagraph();
      closeLists();
      const level = heading[1].length;
      html.push(`<h${level}>${inlineFormat(heading[2])}</h${level}>`);
      continue;
    }
    const tableSep = /^\|?\s*:?-{3,}/.test(line);
    if (line.trim().startsWith('|') && !tableSep) {
      // Collect simple tables as preformatted blocks when consecutive
      flushParagraph();
      closeLists();
      html.push(`<p class="blog-table-line">${inlineFormat(line)}</p>`);
      continue;
    }
    if (tableSep) continue;
    const ul = line.match(/^\s*[-*]\s+(.*)$/);
    if (ul) {
      flushParagraph();
      if (inOl) {
        html.push('</ol>');
        inOl = false;
      }
      if (!inUl) {
        html.push('<ul>');
        inUl = true;
      }
      html.push(`<li>${inlineFormat(ul[1])}</li>`);
      continue;
    }
    const ol = line.match(/^\s*\d+\.\s+(.*)$/);
    if (ol) {
      flushParagraph();
      if (inUl) {
        html.push('</ul>');
        inUl = false;
      }
      if (!inOl) {
        html.push('<ol>');
        inOl = true;
      }
      html.push(`<li>${inlineFormat(ol[1])}</li>`);
      continue;
    }
    closeLists();
    paragraph.push(line.trim());
  }
  flushParagraph();
  closeLists();
  if (inCode) {
    html.push(`<pre><code>${escapeHtml(codeBuffer.join('\n'))}</code></pre>`);
  }
  return html.join('\n');
}

function inlineFormat(text = '') {
  let value = escapeHtml(text);
  value = value.replace(/\[([^\]]+)\]\((https?:[^)\s]+|\/[^)\s]*)\)/g, '<a href="$2" rel="noopener">$1</a>');
  value = value.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  value = value.replace(/`([^`]+)`/g, '<code>$1</code>');
  return value;
}

function escapeHtml(value = '') {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');
}
