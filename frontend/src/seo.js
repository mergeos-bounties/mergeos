const siteName = 'MergeOS';
const defaultOrigin = 'https://mergeos.dev';
const defaultImagePath = '/favicon.svg';
const mergeIdeRepositorySlug = 'mergeos-bounties/mergeos';
const mergeIdeReleaseTag = 'mergeide-windows-latest';
const mergeIdeDownloadFileName = 'MergeIDE-Windows-x64.exe';
const mergeIdeWindowsDownloadUrl = `https://github.com/${mergeIdeRepositorySlug}/releases/download/${mergeIdeReleaseTag}/${mergeIdeDownloadFileName}`;

const publicSeoPaths = {
  home: '/',
  system: '/system',
  customers: '/customers',
  agents: '/agents',
  contributors: '/contributors',
  sdk: '/sdk',
  backend: '/backend',
  admins: '/admins',
  product: '/product',
  solutions: '/solutions',
  marketplace: '/marketplace',
  live: '/live-feed',
  'how-it-works': '/how-it-works',
  ledger: '/ledger',
  protocol: '/protocol',
  contracts: '/contracts',
  airdrop: '/airdrop',
  presale: '/presale',
  whitepaper: '/whitepaper',
  mergeide: '/mergeide',
  terms: '/terms',
  privacy: '/privacy',
};

const publicSeoAliases = {
  system: ['/architecture', '/vision', '/system-vision', '/repository-architecture', '/repo-architecture', '/system-repositories', '/mergeos-app'],
  customers: ['/saas-teams', '/founders', '/repo-owners', '/customer-dashboard'],
  agents: ['/ai-agents', '/agent-lanes', '/ai-layer', '/ai-orchestration', '/ai-workflow', '/repo-scan-ai'],
  contributors: ['/builders', '/workers', '/talent', '/human-contributors'],
  sdk: ['/mergeos-sdk', '/integrations', '/api-sdk', '/developer-sdk'],
  backend: ['/backend-apis', '/backend-system', '/orchestration', '/websocket-gateway', '/task-engine'],
  admins: ['/admin-console', '/admin-ops', '/treasury-ops', '/payout-management', '/moderation'],
  marketplace: ['/work-marketplace', '/developer-marketplace', '/live-projects', '/public-bounties', '/marketplace-bounties', '/ai-agent-marketplace'],
  live: ['/live', '/realtime-feed', '/live-prs', '/deployment-feed', '/ai-action-feed', '/activity-feed', '/payout-feed'],
  ledger: ['/ledger-logs', '/public-ledger', '/proof-ledger', '/escrow-events', '/payout-logs', '/token-mint-logs', '/release-logs', '/ai-action-logs', '/pr-proof-logs'],
  protocol: ['/protocol-index', '/open-protocol', '/mergeos-protocol', '/protocol-roadmap'],
  contracts: ['/mrg', '/token-economy', '/contracts-and-escrow', '/mergeos-contracts', '/escrow-contracts', '/payout-contracts'],
  airdrop: ['/mrg-airdrop', '/task-airdrop', '/airdrop-tasks', '/claim-mrg'],
  presale: ['/mrg-presale', '/token-presale', '/presale-register', '/reserve-mrg'],
  whitepaper: ['/mergeos-whitepaper', '/white-paper', '/paper', '/architecture-paper'],
  mergeide: ['/ide', '/merge-ide', '/download'],
};

const pageSeo = {
  home: {
    title: 'MergeOS | AI software delivery OS with escrow, agents, and ledger proof',
    description: 'MergeOS turns funded software work into verified delivery with repo-aware tasks, AI agents, marketplace routing, escrow payments, MRG token logs, and public ledger proof.',
    keywords: ['AI software delivery', 'developer marketplace', 'escrow software projects', 'AI coding agents', 'public ledger proof'],
  },
  system: {
    title: 'MergeOS System Vision | AI software delivery architecture',
    description: 'Read the MergeOS system vision for AI orchestration, contributor marketplace, escrow payments, PR monitoring, MRG token economy, public ledger logs, SDK, contracts, and open protocol architecture.',
    keywords: ['MergeOS architecture', 'AI software delivery OS', 'AI orchestration', 'developer marketplace architecture', 'MRG token economy', 'public ledger logs'],
  },
  customers: {
    title: 'MergeOS Customers | Project dashboard, escrow, PRs, AI logs, and proof',
    description: 'See how founders, SaaS teams, startups, and repo owners use MergeOS to turn briefs and repositories into funded tasks, live PR monitoring, escrow payments, AI logs, release approvals, and public ledger proof.',
    keywords: ['customer software dashboard', 'SaaS project escrow', 'repo owner workflow', 'live PR monitoring', 'AI project delivery', 'customer ledger proof'],
  },
  agents: {
    title: 'MergeOS AI Layer | Repo scan, task generation, PR review, and deployment validation',
    description: 'Explore the MergeOS AI layer for repository scans, issue analysis, task generation, reward estimation, contributor routing, PR review, testing, security checks, deployment validation, and ledger proof.',
    keywords: ['MergeOS AI layer', 'repo scan AI', 'AI issue analysis', 'AI task generation', 'AI reward estimation', 'AI PR review', 'AI deployment validation'],
  },
  contributors: {
    title: 'MergeOS Contributors | Funded software bounties, reputation, and proof',
    description: 'Explore MergeOS contributor lanes for frontend, backend, design, QA, DevOps, and security work with claimable bounties, PR evidence, escrow-backed rewards, reputation, and public ledger proof.',
    keywords: ['software contributors', 'developer bounties', 'frontend developers', 'backend developers', 'QA engineers', 'DevOps bounties', 'security auditors'],
  },
  sdk: {
    title: 'MergeOS SDK | Task APIs, workflow clients, events, and integrations',
    description: 'Build with the MergeOS SDK for task APIs, workflow APIs, realtime event helpers, webhooks, agent context URLs, evidence submission, and public ledger proof references.',
    keywords: ['MergeOS SDK', 'task API', 'workflow API', 'developer integrations', 'AI agent API', 'ledger proof API', 'webhook events'],
  },
  backend: {
    title: 'MergeOS Backend & APIs | Auth, orchestration, realtime, escrow, and ledger',
    description: 'Explore the MergeOS backend system for authentication, repository imports, AI orchestration, task APIs, payment verification, escrow coordination, WebSocket events, notifications, and ledger proof.',
    keywords: ['MergeOS backend', 'software delivery API', 'AI orchestration backend', 'task engine API', 'WebSocket gateway', 'escrow API', 'ledger proof API'],
  },
  admins: {
    title: 'MergeOS Admins | Treasury, disputes, payouts, moderation, and audit proof',
    description: 'Explore the MergeOS admin operations layer for treasury operators, dispute handlers, moderation queues, payout approvals, fraud signals, contract references, and public ledger audit trails.',
    keywords: ['MergeOS admins', 'admin console', 'treasury operations', 'payout management', 'dispute handling', 'moderation workflow', 'ledger audit proof'],
  },
  product: {
    title: 'MergeOS Product | Escrow-backed project delivery and proof ledger',
    description: 'Explore the MergeOS product workflow for project intake, repo import, AI task graphs, escrow funding, PR monitoring, deployment gates, payouts, and ledger proof.',
    keywords: ['software project escrow', 'repo issue scanner', 'AI task graph', 'PR monitoring', 'deployment proof'],
  },
  solutions: {
    title: 'MergeOS Solutions | Human, AI, and hybrid software delivery lanes',
    description: 'Choose MergeOS delivery lanes for SaaS teams, founders, repo owners, contributors, AI agents, security QA, DevOps, ops, and payouts.',
    keywords: ['software delivery solutions', 'AI agent workflow', 'hybrid delivery', 'developer bounties', 'DevOps proof'],
  },
  marketplace: {
    title: 'MergeOS Marketplace System | Live projects, public bounties, and AI agents',
    description: 'Browse the MergeOS realtime marketplace system for live funded projects, public bounty tasks, contributor signals, AI agent lanes, reward pools, escrow-backed work, and ledger proof.',
    keywords: ['MergeOS marketplace system', 'live software projects', 'public bounties', 'AI agent marketplace', 'developer marketplace', 'escrow-backed rewards', 'contributor signals'],
  },
  live: {
    title: 'MergeOS Live Feed System | Realtime PRs, deployments, AI actions, and payouts',
    description: 'Watch the MergeOS realtime live feed for live PRs, deployments, active contributors, AI actions, task events, escrow changes, payout releases, and ledger-backed proof.',
    keywords: ['MergeOS live feed', 'realtime software delivery', 'live PR feed', 'deployment feed', 'AI action feed', 'active contributors', 'payout events'],
  },
  'how-it-works': {
    title: 'How MergeOS Works | From repo scan to funded delivery proof',
    description: 'Learn how MergeOS imports repositories, scans issues, generates task graphs, funds escrow, routes contributors or agents, reviews PRs, deploys, and releases payouts.',
    keywords: ['how MergeOS works', 'repo scan workflow', 'task generation', 'escrow workflow', 'verified delivery'],
  },
  ledger: {
    title: 'MergeOS Ledger Logs | Public payouts, escrow events, PR proof, AI actions, and releases',
    description: 'Inspect MergeOS public ledger logs for payouts, escrow events, PR proof, AI actions, releases, payment verification, MRG token minting, contract references, and hash-chain proof.',
    keywords: ['MergeOS ledger logs', 'public ledger logs', 'escrow events', 'payout logs', 'PR proof logs', 'AI action logs', 'release logs', 'token mint logs'],
  },
  protocol: {
    title: 'MergeOS Protocol Index | Public schemas, endpoints, and agent context',
    description: 'Discover MergeOS public protocol schemas, API endpoints, realtime events, task manifests, workflow graphs, integration context, and agent runbook URLs.',
    keywords: ['MergeOS protocol', 'protocol index', 'public API schemas', 'agent context URLs', 'workflow protocol'],
  },
  contracts: {
    title: 'MergeOS Contracts and MRG | Token economy, escrow reserve, treasury, and payouts',
    description: 'Track the MergeOS contract-facing economy for MRG token supply, escrow reserves, treasury balances, payout contracts, hash roots, and ledger proof.',
    keywords: ['MRG token', 'escrow contracts', 'token economy', 'treasury proof', 'payout contracts'],
  },
  airdrop: {
    title: 'MergeOS Airdrop | Task-based MRG rewards with public proof',
    description: 'Complete MergeOS airdrop missions through repository imports, bounty work, PR evidence, QA checks, AI agent reviews, and ledger-backed proof before claiming MRG allocation.',
    keywords: ['MergeOS airdrop', 'MRG airdrop', 'task airdrop', 'developer rewards', 'bounty proof', 'ledger proof'],
  },
  presale: {
    title: 'MergeOS Presale | MRG reserve workflow, Solana token path, and ledger receipts',
    description: 'Register interest in the MergeOS MRG presale with wallet readiness, allocation reserve steps, payment verification, Solana contract checkpoints, and public ledger receipts.',
    keywords: ['MRG presale', 'MergeOS presale', 'Solana token presale', 'token reserve', 'presale ledger receipt', 'MRG token'],
  },
  whitepaper: {
    title: 'MergeOS Whitepaper | AI software delivery OS architecture and MRG economy',
    description: 'Read the MergeOS whitepaper for system vision, repository architecture, AI orchestration, marketplace workflow, escrow economy, MRG token model, ledger proof, SDK, and protocol roadmap.',
    keywords: ['MergeOS whitepaper', 'AI software delivery OS', 'MRG token whitepaper', 'software delivery architecture', 'AI agent marketplace', 'open protocol'],
  },
  mergeide: {
    title: 'MergeIDE | Windows exe for MergeOS agents, task packets, and proof',
    description: 'Download the MergeIDE Windows executable for MergeOS delivery, combining task packets, AI agent runbooks, PR review, deployment evidence, ledger proof, and SDK context.',
    keywords: ['MergeIDE', 'MergeIDE Windows exe', 'AI IDE', 'repo-aware IDE', 'agent runbooks', 'task packet workspace', 'developer IDE'],
    type: 'SoftwareApplication',
  },
  terms: {
    title: 'MergeOS Terms of Service | Funded software delivery rules',
    description: 'Read the MergeOS terms for customers, contributors, AI agents, escrow workflows, task claims, reviews, ledger proof, disputes, and payout releases.',
    keywords: ['MergeOS terms', 'software delivery terms', 'escrow rules', 'payout rules'],
  },
  privacy: {
    title: 'MergeOS Privacy Policy | Delivery data, account privacy, and public proof',
    description: 'Learn how MergeOS separates private account, payment, repository, and workspace data from public proof, marketplace, and ledger transparency records.',
    keywords: ['MergeOS privacy', 'delivery data privacy', 'public proof privacy', 'repository data'],
  },
};

function normalizeSeoPath(path = '/') {
  const pathname = String(path || '/').split('?')[0].split('#')[0] || '/';
  const normalized = pathname.replace(/\/+$/, '') || '/';
  return normalized.startsWith('/') ? normalized : `/${normalized}`;
}

export function seoPageFromPath(path = '/') {
  const normalized = normalizeSeoPath(path);
  const direct = Object.entries(publicSeoPaths).find(([, route]) => route === normalized);
  if (direct) return direct[0];
  const alias = Object.entries(publicSeoAliases).find(([, aliases]) =>
    aliases.some((route) => normalizeSeoPath(route) === normalized),
  );
  return alias?.[0] || 'home';
}

export function seoPathForPage(page = 'home') {
  return publicSeoPaths[page] || publicSeoPaths.home;
}

function absoluteUrl(path = '/', origin = defaultOrigin) {
  const base = String(origin || defaultOrigin).replace(/\/+$/, '') || defaultOrigin;
  if (/^https?:\/\//i.test(path)) return path;
  return `${base}${String(path || '/').startsWith('/') ? path : `/${path}`}`;
}

function escapeHtml(value = '') {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');
}

function safeJsonLd(value) {
  return JSON.stringify(value).replaceAll('<', '\\u003c');
}

export function getSeoDataForPath(path = '/', options = {}) {
  const page = options.page || seoPageFromPath(path);
  const entry = pageSeo[page] || pageSeo.home;
  const routePath = seoPathForPage(page);
  const origin = options.origin || defaultOrigin;
  const canonical = absoluteUrl(routePath, origin);
  const image = absoluteUrl(defaultImagePath, origin);
  const graph = [
    {
      '@type': 'Organization',
      '@id': absoluteUrl('/#organization', origin),
      name: siteName,
      url: absoluteUrl('/', origin),
      logo: image,
    },
    {
      '@type': 'WebSite',
      '@id': absoluteUrl('/#website', origin),
      name: siteName,
      url: absoluteUrl('/', origin),
      publisher: { '@id': absoluteUrl('/#organization', origin) },
    },
    {
      '@type': 'WebPage',
      '@id': `${canonical}#webpage`,
      url: canonical,
      name: entry.title,
      description: entry.description,
      isPartOf: { '@id': absoluteUrl('/#website', origin) },
      about: { '@id': absoluteUrl('/#organization', origin) },
    },
  ];

  if (page === 'mergeide') {
    graph.push({
      '@type': 'SoftwareApplication',
      '@id': `${canonical}#software`,
      name: 'MergeIDE',
      applicationCategory: 'DeveloperApplication',
      operatingSystem: 'Windows',
      description: entry.description,
      url: canonical,
      downloadUrl: mergeIdeWindowsDownloadUrl,
      publisher: { '@id': absoluteUrl('/#organization', origin) },
      offers: {
        '@type': 'Offer',
        price: '0',
        priceCurrency: 'USD',
        availability: 'https://schema.org/InStock',
      },
    });
  }

  if (page === 'airdrop') {
    graph.push({
      '@type': 'ItemList',
      '@id': `${canonical}#airdrop-missions`,
      name: 'MergeOS task-based airdrop missions',
      description: entry.description,
      itemListElement: [
        ['Connect account and wallet', 'Use a verified MergeOS account and wallet before claiming allocation.'],
        ['Import repository or start project', 'Attach real software context through project intake or repository import.'],
        ['Complete bounty or agent work', 'Submit accepted task, PR, QA, review, or agent evidence.'],
        ['Publish proof', 'Expose sanitized live feed, ledger, or protocol references before claim.'],
        ['Claim allocation window', 'Eligible proof packets can enter the MRG allocation queue.'],
      ].map(([name, description], index) => ({
        '@type': 'ListItem',
        position: index + 1,
        item: {
          '@type': 'Action',
          name,
          description,
        },
      })),
    });
  }

  if (page === 'presale') {
    graph.push({
      '@type': 'ItemList',
      '@id': `${canonical}#presale-workflow`,
      name: 'MergeOS MRG presale reserve workflow',
      description: entry.description,
      itemListElement: [
        ['Create account', 'Start from an authenticated MergeOS identity.'],
        ['Prepare Solana wallet', 'Attach wallet readiness before distribution.'],
        ['Reserve allocation', 'Review amount, allocation tier, and funding rail.'],
        ['Verify funding', 'Payment or crypto reference must pass review.'],
        ['Publish receipt', 'Accepted reservations should produce ledger-visible receipt and contract references.'],
      ].map(([name, description], index) => ({
        '@type': 'ListItem',
        position: index + 1,
        item: {
          '@type': 'Action',
          name,
          description,
        },
      })),
    });
  }

  if (page === 'whitepaper') {
    graph.push({
      '@type': 'TechArticle',
      '@id': `${canonical}#whitepaper`,
      headline: 'MergeOS whitepaper for AI software delivery architecture and MRG economy',
      description: entry.description,
      url: canonical,
      about: [
        'AI software delivery OS',
        'Repository import',
        'Task graph generation',
        'Contributor marketplace',
        'MRG token economy',
        'Solana contracts',
        'Public ledger proof',
        'Open protocol roadmap',
      ],
      publisher: { '@id': absoluteUrl('/#organization', origin) },
    });
  }

  if (page === 'system') {
    graph.push({
      '@type': 'TechArticle',
      '@id': `${canonical}#architecture`,
      headline: 'MergeOS system architecture and product vision',
      description: entry.description,
      url: canonical,
      about: [
        'AI orchestration',
        'Contributor marketplace',
        'Escrow payments',
        'Pull request monitoring',
        'MRG token economy',
        'Public ledger proof',
        'Open protocol',
      ],
      publisher: { '@id': absoluteUrl('/#organization', origin) },
    });
    graph.push({
      '@type': 'ItemList',
      '@id': `${canonical}#repository-architecture`,
      name: 'MergeOS repository architecture',
      description: 'MergeOS is organized into mergeos-app, mergeos-contracts, mergeos-sdk, and the future mergeos-protocol layer.',
      itemListElement: [
        ['mergeos-app', 'Primary product repository containing frontend, backend, dashboards, APIs, SSR, orchestration logic, realtime feeds, auth, task engine, and ledger-facing application state.'],
        ['mergeos-contracts', 'Blockchain and treasury repository for MRG token, escrow contracts, treasury contracts, payout contracts, contract roots, reserves, and proof anchors.'],
        ['mergeos-sdk', 'Integration and agent repository for task APIs, workflow APIs, event APIs, helper libraries, webhook clients, ledger references, and agent context utilities.'],
        ['mergeos-protocol', 'Future open protocol layer for decentralized execution, external AI agents, public integrations, task manifests, workflow graphs, and ecosystem automation.'],
      ].map(([name, description], index) => ({
        '@type': 'ListItem',
        position: index + 1,
        item: {
          '@type': 'SoftwareSourceCode',
          name,
          description,
          creator: { '@id': absoluteUrl('/#organization', origin) },
          isAccessibleForFree: true,
        },
      })),
    });
  }

  if (page === 'protocol') {
    graph.push({
      '@type': 'TechArticle',
      '@id': `${canonical}#protocol-index`,
      headline: 'MergeOS protocol index for schemas, endpoints, events, and agent context',
      description: entry.description,
      url: canonical,
      about: [
        'Open protocol schemas',
        'Public API endpoints',
        'Realtime WebSocket events',
        'Agent context URLs',
        'Repository artifacts',
        'SDK integration runbooks',
      ],
      publisher: { '@id': absoluteUrl('/#organization', origin) },
    });
    graph.push({
      '@type': 'ItemList',
      '@id': `${canonical}#repository-artifacts`,
      name: 'MergeOS protocol repository artifacts',
      description: 'Repository and artifact index for mergeos-app, mergeos-contracts, mergeos-sdk, and mergeos-protocol.',
      itemListElement: [
        ['mergeos-app', 'Product OS repository for frontend, backend, dashboards, SSR, auth, task engine, realtime feeds, and public pages.', '/system'],
        ['mergeos-contracts', 'Contract and treasury repository for MRG token, escrow, treasury, payout contracts, reserve proof, and ledger-facing roots.', '/contracts'],
        ['mergeos-sdk', 'Integration and agent repository for task APIs, workflow clients, event helpers, WebSocket helpers, and context URLs.', '/sdk'],
        ['mergeos-protocol', 'Open protocol artifact for schemas, endpoint discovery, realtime metadata, agent context URLs, and runbook order.', '/protocol'],
      ].map(([name, description, url], index) => ({
        '@type': 'ListItem',
        position: index + 1,
        item: {
          '@type': 'SoftwareSourceCode',
          name,
          description,
          url: absoluteUrl(url, origin),
          creator: { '@id': absoluteUrl('/#organization', origin) },
          isAccessibleForFree: true,
        },
      })),
    });
  }

  if (page === 'customers') {
    graph.push({
      '@type': 'ItemList',
      '@id': `${canonical}#customer-workflows`,
      name: 'MergeOS customer workflows',
      description: entry.description,
      itemListElement: [
        ['Project overview', 'Customer dashboard for projects, budgets, open tasks, AI activity, and release readiness.'],
        ['Repository import', 'Repo owner workflow for issue scans, technical debt, dependencies, and bounty task generation.'],
        ['Escrow payments', 'Verified funding before contributors or AI agents can claim delivery work.'],
        ['Live PR monitoring', 'Pull request, test, deployment, review, and acceptance evidence tracking.'],
        ['Ledger proof', 'Public escrow, payout, PR, deployment, token mint, and release references.'],
      ].map(([name, description], index) => ({
        '@type': 'ListItem',
        position: index + 1,
        item: {
          '@type': 'Service',
          name,
          description,
          provider: { '@id': absoluteUrl('/#organization', origin) },
        },
      })),
    });
  }

  if (page === 'agents') {
    graph.push({
      '@type': 'TechArticle',
      '@id': `${canonical}#ai-layer`,
      headline: 'MergeOS AI layer for repository-to-delivery orchestration',
      description: entry.description,
      url: canonical,
      about: [
        'Repository scanning',
        'Issue analysis',
        'Task generation',
        'Reward estimation',
        'Contributor routing',
        'Pull request review',
        'Deployment validation',
      ],
      publisher: { '@id': absoluteUrl('/#organization', origin) },
    });
    graph.push({
      '@type': 'ItemList',
      '@id': `${canonical}#agent-lanes`,
      name: 'MergeOS AI agent lanes',
      description: entry.description,
      itemListElement: [
        ['Coding agents', 'Implementation agents for scoped task packets and repository context.'],
        ['Review agents', 'Pull request review agents for correctness, regressions, and acceptance criteria.'],
        ['Testing agents', 'QA agents for unit, integration, accessibility, and smoke evidence.'],
        ['Security agents', 'Security agents for dependency, secret, auth, and risky code checks.'],
        ['Deployment agents', 'Deployment agents for rollout health, preview state, and release gates.'],
      ].map(([name, description], index) => ({
        '@type': 'ListItem',
        position: index + 1,
        item: {
          '@type': 'Service',
          name,
          description,
          provider: { '@id': absoluteUrl('/#organization', origin) },
        },
      })),
    });
    graph.push({
      '@type': 'ItemList',
      '@id': `${canonical}#ai-workflow`,
      name: 'MergeOS AI workflow',
      description: 'Repository import, issue scanning, task generation, reward estimation, routing, PR review, and deployment validation in the MergeOS AI layer.',
      itemListElement: [
        ['Import repository', 'Load repository URLs, imported issues, dependencies, technical debt markers, and task history.'],
        ['Issue scan', 'Detect bugs, technical debt, dependency signals, security exposure, and deployment constraints.'],
        ['Task generation', 'Create scoped task packets with acceptance criteria, required evidence, dependencies, and suggested lane.'],
        ['Reward estimation', 'Estimate complexity, expected time, budget fit, reward range, and review depth.'],
        ['Contributor routing', 'Route work to human contributors, AI agents, QA, security, DevOps, or hybrid lanes.'],
        ['PR review', 'Inspect pull requests, patch summaries, test evidence, risk notes, and acceptance signals.'],
        ['Deployment validation', 'Verify preview health, release status, environment readiness, rollout notes, and proof references.'],
      ].map(([name, description], index) => ({
        '@type': 'ListItem',
        position: index + 1,
        item: {
          '@type': 'SoftwareSourceCode',
          name,
          description,
          provider: { '@id': absoluteUrl('/#organization', origin) },
        },
      })),
    });
  }

  if (page === 'contributors') {
    graph.push({
      '@type': 'ItemList',
      '@id': `${canonical}#contributor-lanes`,
      name: 'MergeOS contributor lanes',
      description: entry.description,
      itemListElement: [
        ['Frontend developers', 'UI, responsive, accessibility, state, and interaction contributors for funded software work.'],
        ['Backend developers', 'API, persistence, auth, orchestration, payment, and realtime workflow contributors.'],
        ['Design contributors', 'Product flow, UI polish, copy, handoff, and visual QA contributors.'],
        ['QA engineers', 'Testing, browser checks, accessibility, regression, and release evidence contributors.'],
        ['DevOps operators', 'Deployment, environment health, preview, rollout, rollback, and release gate contributors.'],
        ['Security auditors', 'Dependency, secret, auth, unsafe path, and production exposure auditors before payout.'],
      ].map(([name, description], index) => ({
        '@type': 'ListItem',
        position: index + 1,
        item: {
          '@type': 'Service',
          name,
          description,
          provider: { '@id': absoluteUrl('/#organization', origin) },
        },
      })),
    });
  }

  if (page === 'marketplace') {
    graph.push({
      '@type': 'CollectionPage',
      '@id': `${canonical}#marketplace-system`,
      name: 'MergeOS marketplace system',
      description: entry.description,
      url: canonical,
      about: [
        'Live funded projects',
        'Public bounty tasks',
        'AI agent lanes',
        'Contributor signals',
        'Escrow-backed rewards',
        'Ledger proof',
      ],
      publisher: { '@id': absoluteUrl('/#organization', origin) },
    });
    graph.push({
      '@type': 'ItemList',
      '@id': `${canonical}#marketplace-features`,
      name: 'MergeOS realtime marketplace features',
      description: entry.description,
      itemListElement: [
        ['Live Projects', 'Funded projects with customer context, escrow status, task counts, budget, delivery timeline, and ledger-backed proof.', '#marketplace-projects'],
        ['Public Bounties', 'Claimable task rows generated from repository scans, issues, reward estimates, acceptance criteria, and route-ready work packets.', '#marketplace-bounties'],
        ['Contributor Signals', 'Human contributor lanes, reputation, payout history, accepted work, reward totals, risk signals, and live delivery evidence.', '#marketplace-contributor-board'],
        ['AI Agent Lanes', 'Routed coding, review, testing, security, deployment, and generation agents with task queues, capability labels, and reward pools.', '#marketplace-agents'],
        ['Escrow-backed Rewards', 'Verified funding, reward pools, claim status, payout readiness, and payment proof before work is routed.', '#marketplace-bounties'],
        ['Ledger Proof', 'Public escrow, bounty, PR, deployment, payout, token mint, and release references for marketplace transparency.', '#marketplace-benefits'],
      ].map(([name, description, anchor], index) => ({
        '@type': 'ListItem',
        position: index + 1,
        item: {
          '@type': 'Service',
          name,
          description,
          url: `${canonical}${anchor}`,
          provider: { '@id': absoluteUrl('/#organization', origin) },
          areaServed: 'Global',
        },
      })),
    });
    graph.push({
      '@type': 'ItemList',
      '@id': `${canonical}#ai-agent-capability-matrix`,
      name: 'MergeOS AI agent capability matrix',
      description: 'AI agent lanes in the MergeOS marketplace for task generation, implementation, PR review, testing, security validation, and deployment proof.',
      itemListElement: [
        ['Task generation', 'Scan repository issues, technical debt, dependencies, and briefs to create task packets, reward estimates, lane routing, and workflow graph evidence.'],
        ['Code implementation', 'Use scoped task packets and repository context to produce branches, patches, pull requests, and commit references.'],
        ['PR review', 'Inspect pull request diffs, acceptance criteria, regressions, risk notes, and release readiness before payout.'],
        ['Testing and QA', 'Run unit, integration, browser, accessibility, and smoke checks with test logs, screenshots, and pass or fail evidence.'],
        ['Security validation', 'Check dependencies, secrets, authentication changes, payment paths, token usage, and risky code before release.'],
        ['Deployment gate', 'Validate preview health, environment readiness, rollout notes, release state, and public deployment proof rows.'],
      ].map(([name, description], index) => ({
        '@type': 'ListItem',
        position: index + 1,
        item: {
          '@type': 'Service',
          name,
          description,
          provider: { '@id': absoluteUrl('/#organization', origin) },
          areaServed: 'Global',
        },
      })),
    });
  }

  if (page === 'live') {
    graph.push({
      '@type': 'CollectionPage',
      '@id': `${canonical}#live-feed-system`,
      name: 'MergeOS live feed system',
      description: entry.description,
      url: canonical,
      about: [
        'Live pull requests',
        'Deployment validation',
        'Active contributors',
        'AI actions',
        'Task events',
        'Escrow and payout proof',
      ],
      publisher: { '@id': absoluteUrl('/#organization', origin) },
    });
    graph.push({
      '@type': 'ItemList',
      '@id': `${canonical}#realtime-proof-lanes`,
      name: 'MergeOS realtime proof lanes',
      description: entry.description,
      itemListElement: [
        ['Live PRs', 'Review submissions, accepted pull requests, task evidence, contributor notes, and public PR references.'],
        ['Deployments', 'Release checks, deployment validation, QA handoff, environment status, rollout updates, and release gates.'],
        ['Active Contributors', 'Human builders and agent lanes surfaced from accepted work, current activity, role, and delivery evidence.'],
        ['AI Actions', 'AI review webhooks, agent action packets, routing signals, risk notes, and validation events.'],
        ['Escrow Events', 'Funding, escrow status, payment verification, token minting, reserve movement, and payout readiness.'],
        ['Payout Releases', 'Accepted task payouts, auto-release events, release references, and ledger-backed proof rows.'],
      ].map(([name, description], index) => ({
        '@type': 'ListItem',
        position: index + 1,
        item: {
          '@type': 'Event',
          name,
          description,
          eventAttendanceMode: 'https://schema.org/OnlineEventAttendanceMode',
          eventStatus: 'https://schema.org/EventScheduled',
          organizer: { '@id': absoluteUrl('/#organization', origin) },
        },
      })),
    });
    graph.push({
      '@type': 'ItemList',
      '@id': `${canonical}#live-event-contracts`,
      name: 'MergeOS realtime event contracts',
      description: 'Protocol and ledger context for live PR, deployment, AI action, and payout event streams.',
      itemListElement: [
        ['PR review stream', 'mergeos.pr.monitor.v1', 'Review submissions and accepted work mapped to task evidence and public PR references.'],
        ['Deployment stream', 'mergeos.deployment.v1', 'Deployment validation and status updates mapped to release gates and environment evidence.'],
        ['AI action stream', 'mergeos.agent.action.v1', 'AI review webhooks and agent action packets mapped to routing, review, testing, and risk signals.'],
        ['Payout and ledger stream', 'mergeos.ledger.events.v1', 'Escrow, token mint, release, and payout rows mapped to ledger-backed public proof.'],
      ].map(([name, version, description], index) => ({
        '@type': 'ListItem',
        position: index + 1,
        item: {
          '@type': 'TechArticle',
          name,
          headline: `${name} contract`,
          description,
          version,
          url: `${canonical}#live-event-contracts`,
          publisher: { '@id': absoluteUrl('/#organization', origin) },
        },
      })),
    });
  }

  if (page === 'ledger') {
    graph.push({
      '@type': 'CollectionPage',
      '@id': `${canonical}#ledger-logs`,
      name: 'MergeOS public ledger logs',
      description: entry.description,
      url: canonical,
      about: [
        'Payout logs',
        'Escrow events',
        'Pull request proof',
        'AI action logs',
        'Release events',
        'MRG token minting',
        'Hash-chain proof',
      ],
      publisher: { '@id': absoluteUrl('/#organization', origin) },
    });
    graph.push({
      '@type': 'ItemList',
      '@id': `${canonical}#ledger-transparency-rows`,
      name: 'MergeOS ledger transparency rows',
      description: entry.description,
      itemListElement: [
        ['Payout logs', 'Task payouts, manual credits, auto-release payouts, worker references, acceptance status, and release proofs.'],
        ['Escrow events', 'Payment verification, project reserves, task reserves, remaining balances, treasury movement, and release readiness.'],
        ['PR proof', 'Pull request handoffs, review submissions, accepted work, deployment evidence, and sanitized public references.'],
        ['AI actions', 'AI reviews, agent actions, routing decisions, risk notes, validation packets, and review webhook evidence.'],
        ['Release events', 'Customer approvals, accepted tasks, deployment gates, payout release references, and public release markers.'],
        ['MRG token minting', 'Verified funding, token mint events, reserve accounting, contract references, and token economy flows.'],
        ['Hash-chain proof', 'Root hashes, public root hashes, verified row counts, contract anchors, and tamper-evident proof manifests.'],
      ].map(([name, description], index) => ({
        '@type': 'ListItem',
        position: index + 1,
        item: {
          '@type': 'Dataset',
          name,
          description,
          creator: { '@id': absoluteUrl('/#organization', origin) },
          isAccessibleForFree: true,
        },
      })),
    });
  }

  if (page === 'sdk') {
    graph.push({
      '@type': 'TechArticle',
      '@id': `${canonical}#sdk`,
      headline: 'MergeOS SDK integration surfaces',
      description: entry.description,
      url: canonical,
      about: [
        'Task APIs',
        'Workflow APIs',
        'Realtime events',
        'Webhook integrations',
        'Agent context URLs',
        'Evidence submission',
        'Ledger proof references',
      ],
      publisher: { '@id': absoluteUrl('/#organization', origin) },
    });
    graph.push({
      '@type': 'ItemList',
      '@id': `${canonical}#sdk-surfaces`,
      name: 'MergeOS SDK surfaces',
      description: entry.description,
      itemListElement: [
        ['Task API clients', 'Create, claim, route, submit, review, and release MergeOS work packets.'],
        ['Workflow APIs', 'Fetch project workflow graphs and route work to contributor or AI agent lanes.'],
        ['Realtime event helpers', 'Subscribe to project, PR, deployment, AI action, payout, and ledger events.'],
        ['Webhook surfaces', 'Receive task claim, review, payout, repo scan, and deployment gate events.'],
        ['Ledger references', 'Resolve public proof hashes, escrow references, token mint rows, and payout logs.'],
      ].map(([name, description], index) => ({
        '@type': 'ListItem',
        position: index + 1,
        item: {
          '@type': 'SoftwareSourceCode',
          name,
          description,
          provider: { '@id': absoluteUrl('/#organization', origin) },
        },
      })),
    });
  }

  if (page === 'backend') {
    graph.push({
      '@type': 'TechArticle',
      '@id': `${canonical}#backend-system`,
      headline: 'MergeOS backend orchestration and API control plane',
      description: entry.description,
      url: canonical,
      about: [
        'Authentication',
        'Repository import',
        'AI orchestration',
        'Task engine',
        'Payment verification',
        'Escrow coordination',
        'WebSocket gateway',
        'Ledger proof',
      ],
      publisher: { '@id': absoluteUrl('/#organization', origin) },
    });
    graph.push({
      '@type': 'ItemList',
      '@id': `${canonical}#backend-capabilities`,
      name: 'MergeOS backend capabilities',
      description: entry.description,
      itemListElement: [
        ['Authentication and sessions', 'Account, OAuth, wallet, role, session, and dashboard access control.'],
        ['Repository import', 'GitHub repositories, imported issues, dependency hints, technical debt, and source context.'],
        ['AI orchestration', 'Issue scans, task graph generation, reward estimation, contributor routing, and agent work packets.'],
        ['Payment and escrow', 'Payment verification, escrow reserves, release readiness, payout queues, and token accounting.'],
        ['Realtime and ledger proof', 'WebSocket events, notifications, public protocol documents, and sanitized ledger references.'],
      ].map(([name, description], index) => ({
        '@type': 'ListItem',
        position: index + 1,
        item: {
          '@type': 'SoftwareSourceCode',
          name,
          description,
          provider: { '@id': absoluteUrl('/#organization', origin) },
        },
      })),
    });
  }

  if (page === 'admins') {
    graph.push({
      '@type': 'ItemList',
      '@id': `${canonical}#admin-operations`,
      name: 'MergeOS admin operations',
      description: entry.description,
      itemListElement: [
        ['Treasury console', 'Escrow reserve, token mint, payout health, and treasury balance oversight.'],
        ['Dispute handling', 'Scope, PR evidence, acceptance note, review outcome, and payout readiness review.'],
        ['Payout approvals', 'Approval, pause, rejection, and audit records for payout exceptions.'],
        ['Moderation queue', 'Unsafe project, spam, suspicious claim, and marketplace abuse moderation.'],
        ['Ledger anchoring', 'Sanitized proof hash, escrow, token, payout, and contract reference audit trails.'],
      ].map(([name, description], index) => ({
        '@type': 'ListItem',
        position: index + 1,
        item: {
          '@type': 'Service',
          name,
          description,
          provider: { '@id': absoluteUrl('/#organization', origin) },
        },
      })),
    });
  }

  return {
    page,
    path: routePath,
    title: entry.title,
    description: entry.description,
    keywords: entry.keywords || [],
    canonical,
    image,
    robots: 'index, follow, max-image-preview:large',
    locale: options.locale || 'en_US',
    type: page === 'mergeide' ? 'product' : 'website',
    structuredData: {
      '@context': 'https://schema.org',
      '@graph': graph,
    },
  };
}

export function renderSeoHead(path = '/', options = {}) {
  const seo = getSeoDataForPath(path, options);
  const keywordContent = seo.keywords.join(', ');
  const alternateLinks = [
    ['en', seo.canonical],
    ['vi', seo.canonical],
    ['zh-Hans', seo.canonical],
    ['ja', seo.canonical],
    ['ko', seo.canonical],
    ['x-default', seo.canonical],
  ];

  return [
    `<title data-mergeos-seo="title">${escapeHtml(seo.title)}</title>`,
    `<meta data-mergeos-seo="description" name="description" content="${escapeHtml(seo.description)}" />`,
    `<meta data-mergeos-seo="keywords" name="keywords" content="${escapeHtml(keywordContent)}" />`,
    `<meta data-mergeos-seo="robots" name="robots" content="${escapeHtml(seo.robots)}" />`,
    `<link data-mergeos-seo="canonical" rel="canonical" href="${escapeHtml(seo.canonical)}" />`,
    ...alternateLinks.map(([lang, href]) => `<link data-mergeos-seo="alternate" rel="alternate" hreflang="${escapeHtml(lang)}" href="${escapeHtml(href)}" />`),
    `<meta data-mergeos-seo="og:site_name" property="og:site_name" content="${siteName}" />`,
    `<meta data-mergeos-seo="og:type" property="og:type" content="${escapeHtml(seo.type)}" />`,
    `<meta data-mergeos-seo="og:title" property="og:title" content="${escapeHtml(seo.title)}" />`,
    `<meta data-mergeos-seo="og:description" property="og:description" content="${escapeHtml(seo.description)}" />`,
    `<meta data-mergeos-seo="og:url" property="og:url" content="${escapeHtml(seo.canonical)}" />`,
    `<meta data-mergeos-seo="og:image" property="og:image" content="${escapeHtml(seo.image)}" />`,
    `<meta data-mergeos-seo="twitter:card" name="twitter:card" content="summary_large_image" />`,
    `<meta data-mergeos-seo="twitter:title" name="twitter:title" content="${escapeHtml(seo.title)}" />`,
    `<meta data-mergeos-seo="twitter:description" name="twitter:description" content="${escapeHtml(seo.description)}" />`,
    `<meta data-mergeos-seo="twitter:image" name="twitter:image" content="${escapeHtml(seo.image)}" />`,
    `<script data-mergeos-seo="jsonld" type="application/ld+json">${safeJsonLd(seo.structuredData)}</script>`,
  ].join('\n    ');
}
