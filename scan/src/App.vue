<template>
  <div class="scan-app">
    <header class="topbar">
      <a class="brand" href="/" @click.prevent="goHome">
        <span class="brand-mark" aria-hidden="true"><img src="/favicon.svg" alt="" /></span>
        <span>
          <strong>MergeOS Scan</strong>
          <small>{{ networkLabel }}</small>
        </span>
      </a>

      <nav class="top-actions" aria-label="Explorer actions">
        <div class="header-account" aria-label="Wallet and GitHub">
          <div class="wallet-address-group">
            <button
              :class="['header-chip', 'wallet-chip', localWalletAddress ? 'ready' : '']"
              type="button"
              :disabled="walletBusy && !localWalletAddress"
              :title="localWalletAddress || 'Create MRG wallet'"
              @click="!localWalletAddress && createGuestWallet()"
            >
              <WalletCards :size="16" />
              <span>
                <small>Wallet</small>
                <strong>{{ localWalletAddress ? shortHash(localWalletAddress, 8, 6) : (walletBusy ? 'Creating...' : 'Create wallet') }}</strong>
              </span>
            </button>
            <button
              v-if="localWalletAddress"
              class="icon-mini wallet-copy-button"
              type="button"
              title="Copy wallet address"
              @click="copyValue(localWalletAddress)"
            >
              <Copy :size="15" />
            </button>
          </div>
          <button
            :class="['header-chip', 'github-chip', githubLinked ? 'connected' : (canLinkGitHub ? 'ready' : '')]"
            type="button"
            :disabled="!canLinkGitHub"
            :title="githubActionTitle"
            @click="canLinkGitHub && startGitHubWalletLink()"
          >
            <GitPullRequest :size="17" />
            <span>
              <small>GitHub</small>
              <strong>{{ githubAccountLabel }}</strong>
            </span>
          </button>
          <button
            v-if="githubLinked"
            class="icon-mini github-copy-button"
            type="button"
            title="Copy GitHub account"
            @click="copyValue(githubAccountLabel)"
          >
            <Copy :size="15" />
          </button>
        </div>
        <button
          :class="['api-nav-link', route.name === 'api' ? 'active' : '']"
          type="button"
          title="Open API docs"
          @click="openApiDocs"
        >
          <Braces :size="18" />
          <span>API</span>
        </button>
        <button class="icon-button" type="button" title="Refresh" @click="loadExplorerData">
          <RefreshCw :size="18" />
        </button>
        <a class="icon-button" title="Open MergeOS" href="https://mergeos.shop/">
          <ExternalLink :size="18" />
        </a>
      </nav>
    </header>

    <main :class="{ 'api-main': route.name === 'api' }">
      <ApiDocs v-if="route.name === 'api'" />
      <template v-else>
        <section class="search-band">
          <div class="search-copy">
            <p>MRG Token Explorer</p>
            <h1>MRG token activity and proof ledger for MergeOS.</h1>
          </div>

          <form class="search-panel" @submit.prevent="submitSearch">
            <Search :size="20" />
            <input
              v-model.trim="searchInput"
              autocomplete="off"
              name="query"
              placeholder="Tx hash, address, block, reference"
            />
            <button type="submit">Search</button>
          </form>
        </section>

        <section class="status-strip" aria-label="Explorer status">
          <article v-for="item in statCards" :key="item.label" class="stat-card">
            <span :class="['stat-icon', item.tone]">
              <component :is="item.icon" :size="18" />
            </span>
            <div>
              <strong>{{ item.value }}</strong>
              <small>{{ item.label }}</small>
            </div>
          </article>
        </section>

        <section v-if="errorMessage" class="notice error">
          <AlertTriangle :size="18" />
          <span>{{ errorMessage }}</span>
          <button type="button" @click="loadExplorerData">Retry</button>
        </section>

        <section v-else-if="loading" class="notice">
          <LoaderCircle :size="18" class="spin" />
          <span>Loading MergeOS ledger...</span>
        </section>

        <section v-else-if="route.name === 'tx'" class="detail-layout">
          <TransactionDetail
            v-if="selectedEntry"
            :entry="selectedEntry"
            :entries="entries"
            :token-symbol="tokenSymbol"
            @copy="copyValue"
            @go-block="openBlock"
            @go-address="openAddress"
            @go-tx="openTx"
          />
          <EmptyState v-else title="Transaction not found" body="No MergeOS ledger entry matches this hash." />
        </section>

        <section v-else-if="route.name === 'address'" class="detail-layout">
          <AddressDetail
            v-if="selectedAddress"
            :address="selectedAddress"
            :entries="addressEntries"
            :token-symbol="tokenSymbol"
            @copy="copyValue"
            @go-tx="openTx"
            @go-address="openAddress"
          />
          <EmptyState v-else title="Address not found" body="No public ledger account matches this address." />
        </section>

        <section v-else-if="route.name === 'block'" class="detail-layout">
          <BlockDetail
            v-if="selectedBlockEntry"
            :entry="selectedBlockEntry"
            :entries="entries"
            :token-symbol="tokenSymbol"
            @copy="copyValue"
            @go-tx="openTx"
            @go-address="openAddress"
          />
          <EmptyState v-else title="Ledger block not found" body="This sequence is not in the public MergeOS ledger." />
        </section>

        <template v-else>
          <section class="dashboard-grid">
            <div class="activity-panel">
              <div class="panel-head">
                <div>
                  <p>Latest Transactions</p>
                  <h2>MRG ledger activity</h2>
                </div>
                <div class="table-tools">
                  <select v-model="typeFilter" aria-label="Transaction type">
                    <option value="all">All types</option>
                    <option v-for="type in ledgerTypes" :key="type" :value="type">{{ typeLabel(type) }}</option>
                  </select>
                  <button class="compact-button" type="button" @click="resetFilters">
                    <RotateCcw :size="16" />
                    Reset
                  </button>
                </div>
              </div>

              <TransactionTable
                :entries="displayedEntries"
                :token-symbol="tokenSymbol"
                @go-tx="openTx"
                @go-block="openBlock"
                @go-address="openAddress"
              />
              <div v-if="hasMoreTransactions || showAllTransactions" class="table-more">
                <button type="button" @click="showAllTransactions = !showAllTransactions">
                  {{ showAllTransactions ? 'Show fewer transactions' : 'View more transactions ->' }}
                </button>
              </div>
            </div>

            <aside class="side-rail">
              <section class="rail-panel">
                <div class="panel-head compact">
                  <div>
                    <p>MRG</p>
                    <h2>Token Profile</h2>
                  </div>
                  <span class="pill">Live</span>
                </div>
                <dl class="ledger-summary-list">
                  <div>
                    <dt>Symbol</dt>
                    <dd>{{ tokenSymbol }}</dd>
                  </div>
                  <div>
                    <dt>Total minted</dt>
                    <dd>{{ formatCompact(stats.mintedTokens) }} {{ tokenSymbol }}</dd>
                  </div>
                  <div>
                    <dt>Verified funding</dt>
                    <dd>{{ formatLedgerAmount(stats.fundingCents) }}</dd>
                  </div>
                  <div>
                    <dt>Payment mode</dt>
                    <dd>{{ paymentMode }}</dd>
                  </div>
                </dl>
              </section>

              <section class="rail-panel about-panel">
                <div class="panel-head compact">
                  <div>
                    <p>About MergeOS</p>
                  </div>
                </div>
                <div class="about-copy">
                  <span class="about-icon" aria-hidden="true">
                    <Keyboard :size="42" />
                  </span>
                  <p>MergeOS is a decentralized ledger system for verifying contributions and rewarding real work onchain.</p>
                </div>
              </section>
            </aside>
          </section>
        </template>
      </template>
    </main>

    <footer class="footer">
      <span>scan.mergeos.shop</span>
      <span>Last sync {{ lastSyncLabel }}</span>
      <a href="/api/public/ledger">Public API</a>
    </footer>
  </div>
</template>

<script setup>
import { computed, defineAsyncComponent, defineComponent, h, onBeforeUnmount, onMounted, ref } from 'vue';
import {
  AlertTriangle,
  ArrowDownLeft,
  ArrowUpRight,
  Braces,
  Coins,
  Copy,
  ExternalLink,
  FileText,
  Fingerprint,
  GitPullRequest,
  Keyboard,
  LoaderCircle,
  RefreshCw,
  RotateCcw,
  Search,
  ShieldCheck,
  Trophy,
  WalletCards,
} from '@lucide/vue';
import {
  accountRole,
  aggregateAccounts,
  buildExplorerStats,
  filterEntries,
  findExplorerTarget,
  formatCompactNumber,
  formatLedgerDate,
  githubProfileURL,
  ledgerTypeMeta,
  normalizeLedgerAccount,
  normalizeExplorerPath,
  parseExplorerRoute,
  paymentModeLabel,
  shortHash,
  sortLedgerEntries,
  tokenAmountFromCents,
} from './explorer.js';

const ApiDocs = defineAsyncComponent(() => import('./ApiDocs.vue'));
const apiBase = String(import.meta.env.VITE_MERGEOS_API_BASE || '').replace(/\/$/, '');
const walletStorageKey = 'mergeos_scan_wallet_address';
const walletRecoveryStorageKey = 'mergeos_scan_wallet_recovery';
const githubUserStorageKey = 'mergeos_scan_github_user';
const loading = ref(true);
const errorMessage = ref('');
const config = ref({});
const rawEntries = ref([]);
const marketplace = ref({ stats: {}, projects: [] });
const localWalletAddress = ref(readStoredValue(walletStorageKey));
const walletRecoveryCode = ref(readStoredValue(walletRecoveryStorageKey));
const walletSummary = ref(null);
const walletBusy = ref(false);
const walletError = ref('');
const githubUser = ref(readStoredJSON(githubUserStorageKey));
const lastSyncAt = ref(null);
const searchInput = ref('');
const queryFilter = ref('');
const typeFilter = ref('all');
const showAllTransactions = ref(false);
const route = ref(parseRoute());

const tokenSymbol = computed(() => config.value?.token_symbol || marketplace.value?.stats?.token_symbol || 'MRG');
const paymentMode = computed(() => paymentModeLabel(config.value?.payment_mode));
const githubOAuthReady = computed(() => Boolean(config.value?.github_oauth_ready && config.value?.github_oauth_client_id));
const networkLabel = computed(() => config.value?.environment === 'production' ? 'MergeOS main ledger' : 'MergeOS ledger');
const linkedGitHubUsername = computed(() => cleanGitHubUsername(
  githubUser.value?.github_username || walletSummary.value?.github_username || githubUser.value?.name || '',
));
const githubLinked = computed(() => Boolean(linkedGitHubUsername.value));
const canLinkGitHub = computed(() => Boolean(localWalletAddress.value && !githubLinked.value && !walletBusy.value));
const githubAccountLabel = computed(() => (githubLinked.value ? `github:${linkedGitHubUsername.value}` : 'Connect'));
const githubActionTitle = computed(() => {
  if (githubLinked.value) return `Linked as github:${linkedGitHubUsername.value}`;
  if (!localWalletAddress.value) return 'Create a wallet first';
  if (!githubOAuthReady.value) return 'GitHub App login is not configured yet';
  return 'Connect GitHub to wallet';
});
const entries = computed(() => sortLedgerEntries(rawEntries.value));
const newestEntries = computed(() => entries.value.slice().reverse());
const accounts = computed(() => aggregateAccounts(entries.value));
const stats = computed(() => buildExplorerStats(entries.value, marketplace.value, tokenSymbol.value));
const ledgerTypes = computed(() => Array.from(new Set(entries.value.map((entry) => entry.type))).sort());
const visibleEntries = computed(() => filterEntries(newestEntries.value, { query: queryFilter.value, type: typeFilter.value }));
const displayedEntries = computed(() => (showAllTransactions.value ? visibleEntries.value : visibleEntries.value.slice(0, 5)));
const hasMoreTransactions = computed(() => visibleEntries.value.length > displayedEntries.value.length);
const lastSyncLabel = computed(() => {
  if (!lastSyncAt.value) return 'pending';
  return lastSyncAt.value.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
});
const selectedEntry = computed(() => {
  if (route.value.name !== 'tx') return null;
  const hash = String(route.value.value || '').toLowerCase();
  return entries.value.find((entry) => entry.entry_hash.toLowerCase() === hash || entry.entry_hash.toLowerCase().startsWith(hash));
});
const selectedAddress = computed(() => {
  if (route.value.name !== 'address') return null;
  const target = String(route.value.value || '').toLowerCase();
  const accountTarget = normalizeLedgerAccount(target).toLowerCase();
  return accounts.value.find((row) => normalizeLedgerAccount(row.account).toLowerCase() === accountTarget);
});
const addressEntries = computed(() => {
  if (!selectedAddress.value) return [];
  return filterEntries(newestEntries.value, { account: selectedAddress.value.account });
});
const selectedBlockEntry = computed(() => {
  if (route.value.name !== 'block') return null;
  const sequence = Number(route.value.value);
  return entries.value.find((entry) => entry.sequence === sequence);
});
const statCards = computed(() => [
  { label: 'Ledger Entries', value: formatCompact(stats.value.totalTransactions), icon: FileText, tone: 'green' },
  { label: 'MRG Minted', value: `${formatCompact(stats.value.mintedTokens)} ${tokenSymbol.value}`, icon: Coins, tone: 'green' },
  { label: 'Verified Funding', value: formatLedgerAmount(stats.value.fundingCents), icon: ShieldCheck, tone: 'green' },
  { label: 'Ledger Height', value: `#${formatCompact(stats.value.chainHeight)}`, icon: Trophy, tone: 'gold' },
]);

onMounted(() => {
  migrateLegacyHashRoute();
  window.addEventListener('popstate', syncRoute);
  void handleGitHubWalletCallback();
  void loadExplorerData();
  void loadLocalWalletSummary();
});

onBeforeUnmount(() => {
  window.removeEventListener('popstate', syncRoute);
});

async function loadExplorerData() {
  loading.value = true;
  errorMessage.value = '';
  try {
    const [configResult, ledgerResult, marketplaceResult] = await Promise.allSettled([
      fetchJSON('/api/config'),
      fetchJSON('/api/public/ledger'),
      fetchJSON('/api/public/marketplace'),
    ]);

    if (configResult.status === 'fulfilled') config.value = configResult.value || {};
    if (marketplaceResult.status === 'fulfilled') {
      marketplace.value = normalizeMarketplace(marketplaceResult.value);
    }
    if (ledgerResult.status !== 'fulfilled') {
      throw ledgerResult.reason;
    }

    rawEntries.value = Array.isArray(ledgerResult.value) ? ledgerResult.value : [];
    lastSyncAt.value = new Date();
  } catch (error) {
    errorMessage.value = error?.message || 'Could not load MergeOS ledger.';
  } finally {
    loading.value = false;
  }
}

async function createGuestWallet() {
  walletBusy.value = true;
  walletError.value = '';
  try {
    const payload = await postJSON('/api/wallets', {});
    localWalletAddress.value = payload.address;
    walletRecoveryCode.value = payload.recovery_code;
    walletSummary.value = payload.wallet || null;
    writeStoredValue(walletStorageKey, localWalletAddress.value);
    writeStoredValue(walletRecoveryStorageKey, walletRecoveryCode.value);
    if (localWalletAddress.value) {
      searchInput.value = localWalletAddress.value;
    }
  } catch (error) {
    walletError.value = error.message || 'Could not create wallet.';
  } finally {
    walletBusy.value = false;
  }
}

async function loadLocalWalletSummary() {
  const address = localWalletAddress.value;
  if (!address) return;
  try {
    walletSummary.value = await fetchJSON(`/api/wallets/${encodeURIComponent(address)}`);
  } catch (error) {
    walletError.value = error.message || 'Could not load wallet.';
  }
}

async function startGitHubWalletLink() {
  walletError.value = '';
  if (!localWalletAddress.value) {
    await createGuestWallet();
  }
  if (!localWalletAddress.value) return;
  if (!config.value?.github_oauth_client_id) {
    try {
      config.value = await fetchJSON('/api/config');
    } catch (error) {
      walletError.value = error.message;
      return;
    }
  }
  if (!githubOAuthReady.value) {
    walletError.value = 'GitHub App login is not configured yet.';
    return;
  }
  const state = randomOAuthState();
  const redirectURI = `${window.location.origin}/`;
  window.sessionStorage.setItem('mergeos_scan_github_state', state);
  window.sessionStorage.setItem('mergeos_scan_github_redirect', redirectURI);
  window.sessionStorage.setItem('mergeos_scan_return_path', window.location.pathname || '/');
  window.sessionStorage.setItem('mergeos_scan_wallet_address', localWalletAddress.value);
  window.sessionStorage.setItem('mergeos_scan_wallet_recovery', walletRecoveryCode.value || '');
  const params = new URLSearchParams({
    client_id: config.value.github_oauth_client_id,
    redirect_uri: redirectURI,
    state,
  });
  window.location.href = `https://github.com/login/oauth/authorize?${params.toString()}`;
}

async function handleGitHubWalletCallback() {
  const params = new URLSearchParams(window.location.search);
  const code = params.get('code');
  const state = params.get('state');
  if (!code) return false;

  const expectedState = window.sessionStorage.getItem('mergeos_scan_github_state') || '';
  const redirectURI = window.sessionStorage.getItem('mergeos_scan_github_redirect') || `${window.location.origin}${window.location.pathname}`;
  const returnPath = safeReturnPath(window.sessionStorage.getItem('mergeos_scan_return_path') || '/');
  const walletAddress = window.sessionStorage.getItem('mergeos_scan_wallet_address') || localWalletAddress.value;
  const recoveryCode = window.sessionStorage.getItem('mergeos_scan_wallet_recovery') || walletRecoveryCode.value;
  window.sessionStorage.removeItem('mergeos_scan_github_state');
  window.sessionStorage.removeItem('mergeos_scan_github_redirect');
  window.sessionStorage.removeItem('mergeos_scan_return_path');
  window.sessionStorage.removeItem('mergeos_scan_wallet_address');
  window.sessionStorage.removeItem('mergeos_scan_wallet_recovery');
  window.history.replaceState(null, '', returnPath);
  route.value = parseRoute();

  if (!expectedState || state !== expectedState) {
    walletError.value = 'GitHub sign-in state did not match. Please try again.';
    return true;
  }

  walletBusy.value = true;
  walletError.value = '';
  try {
    const auth = await postJSON('/api/auth/github', {
      code,
      redirect_uri: redirectURI,
      wallet_address: walletAddress,
      recovery_code: recoveryCode,
    });
    githubUser.value = auth.user || null;
    writeStoredJSON(githubUserStorageKey, githubUser.value);
    if (auth.user?.wallet_address) {
      localWalletAddress.value = auth.user.wallet_address;
      writeStoredValue(walletStorageKey, localWalletAddress.value);
    }
    await loadLocalWalletSummary();
  } catch (error) {
    walletError.value = error.message || 'Could not link GitHub.';
  } finally {
    walletBusy.value = false;
  }
  return true;
}

async function fetchJSON(path) {
  const response = await fetch(`${apiBase}${path}`, {
    headers: { Accept: 'application/json' },
  });
  const text = await response.text();
  let payload = {};
  try {
    payload = text ? JSON.parse(text) : {};
  } catch {
    payload = { error: text || 'Request failed' };
  }
  if (!response.ok) {
    throw new Error(payload.error || `Request failed with ${response.status}`);
  }
  return payload;
}

async function postJSON(path, body = {}) {
  const response = await fetch(`${apiBase}${path}`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
  });
  const text = await response.text();
  let payload = {};
  try {
    payload = text ? JSON.parse(text) : {};
  } catch {
    payload = { error: text || 'Request failed' };
  }
  if (!response.ok) {
    throw new Error(payload.error || `Request failed with ${response.status}`);
  }
  return payload;
}

function readStoredValue(key) {
  try {
    return window.localStorage.getItem(key) || '';
  } catch {
    return '';
  }
}

function writeStoredValue(key, value) {
  try {
    window.localStorage.setItem(key, value || '');
  } catch {
    // Storage can be disabled; the wallet still exists on the backend.
  }
}

function readStoredJSON(key) {
  try {
    return JSON.parse(window.localStorage.getItem(key) || 'null');
  } catch {
    return null;
  }
}

function writeStoredJSON(key, value) {
  try {
    if (value) {
      window.localStorage.setItem(key, JSON.stringify(value));
    } else {
      window.localStorage.removeItem(key);
    }
  } catch {
    // Ignore local storage failures.
  }
}

function cleanGitHubUsername(value = '') {
  return String(value || '').trim().replace(/^github:/i, '').trim();
}

function randomOAuthState() {
  if (window.crypto?.getRandomValues) {
    const bytes = new Uint8Array(16);
    window.crypto.getRandomValues(bytes);
    return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function normalizeMarketplace(payload = {}) {
  return {
    stats: payload.stats || {},
    projects: Array.isArray(payload.projects) ? payload.projects : [],
  };
}

function submitSearch() {
  const query = searchInput.value.trim();
  queryFilter.value = query;
  showAllTransactions.value = false;
  if (!query) {
    goHome();
    return;
  }
  const target = findExplorerTarget(entries.value, accounts.value, query);
  if (!target) {
    route.value = { name: 'home', value: '' };
    history.replaceState(null, '', '/');
    return;
  }
  if (target.kind === 'tx') openTx(target.value);
  if (target.kind === 'block') openBlock(target.value);
  if (target.kind === 'address') openAddress(target.value);
}

function resetFilters() {
  searchInput.value = '';
  queryFilter.value = '';
  typeFilter.value = 'all';
  showAllTransactions.value = false;
}

function openTx(hash) {
  setRoute(`/tx/${encodeURIComponent(hash)}`);
}

function openAddress(address) {
  setRoute(`/address/${encodeURIComponent(address)}`);
}

function openBlock(sequence) {
  setRoute(`/block/${encodeURIComponent(sequence)}`);
}

function goHome() {
  setRoute('/');
}

function openApiDocs() {
  setRoute('/api-docs');
}

function setRoute(path) {
  window.history.pushState(null, '', normalizeExplorerPath(path));
  route.value = parseRoute();
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

function syncRoute() {
  route.value = parseRoute();
}

function parseRoute() {
  if (normalizeExplorerPath(window.location.pathname) === '/api-docs') {
    return { name: 'api', value: '' };
  }
  return parseExplorerRoute(window.location.pathname, window.location.hash);
}

function migrateLegacyHashRoute() {
  const legacyPath = String(window.location.hash || '').replace(/^#/, '');
  if (!legacyPath.startsWith('/')) return;
  window.history.replaceState(null, '', normalizeExplorerPath(legacyPath));
  route.value = parseRoute();
}

function safeReturnPath(path = '/') {
  const normalized = normalizeExplorerPath(path);
  if (normalized.startsWith('//') || normalized.startsWith('/api/') || normalized === '/api') return '/';
  return normalized;
}

async function copyValue(value) {
  try {
    await navigator.clipboard.writeText(String(value || ''));
  } catch {
    const input = document.createElement('textarea');
    input.value = String(value || '');
    document.body.appendChild(input);
    input.select();
    document.execCommand('copy');
    input.remove();
  }
}

function typeLabel(type) {
  return ledgerTypeMeta(type).label;
}

function formatLedgerAmount(cents, symbol = tokenSymbol.value) {
  return `${formatCompactNumber(tokenAmountFromCents(cents))} ${symbol}`;
}

function formatCompact(value) {
  return formatCompactNumber(value);
}

const EmptyState = defineComponent({
  props: {
    title: { type: String, required: true },
    body: { type: String, required: true },
  },
  setup(props) {
    return () => h('div', { class: 'empty-state' }, [
      h(Fingerprint, { size: 34 }),
      h('h2', props.title),
      h('p', props.body),
    ]);
  },
});

const TransactionTable = defineComponent({
  props: {
    entries: { type: Array, required: true },
    tokenSymbol: { type: String, required: true },
  },
  emits: ['go-tx', 'go-block', 'go-address'],
  setup(props, { emit }) {
    return () => h('div', { class: 'table-wrap' }, [
      h('table', { class: 'tx-table' }, [
        h('thead', [
          h('tr', [
            h('th', 'Txn Hash'),
            h('th', 'Block'),
            h('th', 'Type'),
            h('th', 'From'),
            h('th', 'To'),
            h('th', 'Value'),
            h('th', 'Age'),
          ]),
        ]),
        h('tbody', props.entries.length ? props.entries.map((entry) => txRow(entry, props.tokenSymbol, emit)) : [
          h('tr', [h('td', { class: 'state-cell', colspan: 7 }, 'No matching transactions.')]),
        ]),
      ]),
    ]);
  },
});

function txRow(entry, tokenSymbolValue, emit) {
  const meta = ledgerTypeMeta(entry.type);
  const when = formatLedgerDate(entry.created_at);
  return h('tr', { key: entry.entry_hash || entry.sequence }, [
    h('td', [
      h('button', { class: 'link-button hash-link', type: 'button', onClick: () => emit('go-tx', entry.entry_hash) }, shortHash(entry.entry_hash, 10, 8)),
      h('small', entry.reference || '-'),
    ]),
    h('td', [
      h('button', { class: 'link-button block-link', type: 'button', onClick: () => emit('go-block', entry.sequence) }, `#${entry.sequence}`),
    ]),
    h('td', [h('span', { class: ['type-badge', meta.tone] }, meta.label)]),
    h('td', [addressButton(entry.from_account, emit)]),
    h('td', [addressButton(entry.to_account, emit)]),
    h('td', { class: 'value-cell' }, valueLabel(entry, tokenSymbolValue)),
    h('td', when.date),
  ]);
}

function addressButton(account, emit) {
  if (!account) return h('span', '-');
  return h('button', { class: 'link-button address-link', type: 'button', onClick: () => emit('go-address', account) }, shortHash(account, 14, 8));
}

function valueLabel(entry, tokenSymbolValue) {
  return formatLedgerAmount(entry.amount_cents, tokenSymbolValue);
}

const DetailField = defineComponent({
  props: {
    label: { type: String, required: true },
    value: { type: [String, Number], default: '' },
    copyable: { type: Boolean, default: false },
  },
  emits: ['copy'],
  setup(props, { emit }) {
    return () => h('div', { class: 'detail-field' }, [
      h('dt', props.label),
      h('dd', [
        h('span', String(props.value || '-')),
        props.copyable ? h('button', { class: 'icon-mini', type: 'button', title: 'Copy', onClick: () => emit('copy', props.value) }, [h(Copy, { size: 15 })]) : null,
      ]),
    ]);
  },
});

const TransactionDetail = defineComponent({
  props: {
    entry: { type: Object, required: true },
    entries: { type: Array, required: true },
    tokenSymbol: { type: String, required: true },
  },
  emits: ['copy', 'go-block', 'go-address', 'go-tx'],
  setup(props, { emit }) {
    return () => {
      const entry = props.entry;
      const meta = ledgerTypeMeta(entry.type);
      const when = formatLedgerDate(entry.created_at);
      const index = props.entries.findIndex((item) => item.entry_hash === entry.entry_hash);
      const previous = index > 0 ? props.entries[index - 1] : null;
      const next = index >= 0 && index < props.entries.length - 1 ? props.entries[index + 1] : null;
      const chainLinked = index <= 0 || entry.previous_hash === previous?.entry_hash;
      return h('article', { class: 'detail-panel' }, [
        detailHeader('Transaction Details', entry.entry_hash, meta.label, meta.tone, emit),
        h('div', { class: 'detail-grid' }, [
          h(DetailField, { label: 'Transaction Hash', value: entry.entry_hash, copyable: true, onCopy: (value) => emit('copy', value) }),
          h(DetailField, { label: 'Status', value: chainLinked ? 'Verified' : 'Hash chain check required' }),
          h('div', { class: 'detail-field' }, [
            h('dt', 'Block'),
            h('dd', [h('button', { class: 'link-button', type: 'button', onClick: () => emit('go-block', entry.sequence) }, `#${entry.sequence}`)]),
          ]),
          h(DetailField, { label: 'Timestamp', value: when.full }),
          accountField('From', entry.from_account, emit),
          accountField('To', entry.to_account, emit),
          h(DetailField, { label: 'Value', value: valueLabel(entry, props.tokenSymbol) }),
          h(DetailField, { label: 'Reference', value: entry.reference, copyable: true, onCopy: (value) => emit('copy', value) }),
          h(DetailField, { label: 'Previous Hash', value: entry.previous_hash, copyable: true, onCopy: (value) => emit('copy', value) }),
        ]),
        h('div', { class: 'detail-actions' }, [
          previous ? h('button', { type: 'button', onClick: () => emit('go-tx', previous.entry_hash) }, [h(ArrowDownLeft, { size: 16 }), 'Previous Tx']) : null,
          next ? h('button', { type: 'button', onClick: () => emit('go-tx', next.entry_hash) }, [h(ArrowUpRight, { size: 16 }), 'Next Tx']) : null,
        ]),
      ]);
    };
  },
});

const AddressDetail = defineComponent({
  props: {
    address: { type: Object, required: true },
    entries: { type: Array, required: true },
    tokenSymbol: { type: String, required: true },
  },
  emits: ['copy', 'go-tx', 'go-address'],
  setup(props, { emit }) {
    return () => {
      const githubURL = githubProfileURL(props.address.account);
      return h('article', { class: 'detail-panel' }, [
        detailHeader('Address Details', props.address.account, accountRole(props.address.account), 'address', emit),
        h('div', { class: 'address-summary' }, [
          metric('Transactions', props.address.tx_count),
          metric('Received', formatLedgerAmount(props.address.received_cents, props.tokenSymbol)),
          metric('Sent', formatLedgerAmount(props.address.sent_cents, props.tokenSymbol)),
          metric('Net', formatLedgerAmount(props.address.net_cents, props.tokenSymbol)),
        ]),
        githubURL
          ? h('div', { class: 'detail-actions' }, [
              h('a', { href: githubURL, target: '_blank', rel: 'noreferrer' }, [h(ExternalLink, { size: 16 }), `Open ${props.address.account} on GitHub`]),
            ])
          : null,
        h(TransactionTable, { entries: props.entries, tokenSymbol: props.tokenSymbol, onGoTx: (value) => emit('go-tx', value), onGoAddress: (value) => emit('go-address', value), onGoBlock: () => {} }),
      ]);
    };
  },
});

const BlockDetail = defineComponent({
  props: {
    entry: { type: Object, required: true },
    entries: { type: Array, required: true },
    tokenSymbol: { type: String, required: true },
  },
  emits: ['copy', 'go-tx', 'go-address'],
  setup(props, { emit }) {
    return () => {
      const when = formatLedgerDate(props.entry.created_at);
      return h('article', { class: 'detail-panel' }, [
        detailHeader('Ledger Block', `#${props.entry.sequence}`, 'One public proof entry', 'block', emit),
        h('div', { class: 'detail-grid' }, [
          h(DetailField, { label: 'Block Height', value: `#${props.entry.sequence}` }),
          h(DetailField, { label: 'Timestamp', value: when.full }),
          h(DetailField, { label: 'Block Hash', value: props.entry.entry_hash, copyable: true, onCopy: (value) => emit('copy', value) }),
          h(DetailField, { label: 'Parent Hash', value: props.entry.previous_hash, copyable: true, onCopy: (value) => emit('copy', value) }),
        ]),
        h(TransactionTable, { entries: [props.entry], tokenSymbol: props.tokenSymbol, onGoTx: (value) => emit('go-tx', value), onGoAddress: (value) => emit('go-address', value), onGoBlock: () => {} }),
      ]);
    };
  },
});

function detailHeader(title, primary, badge, tone, emit) {
  const showFullPrimary = tone === 'address';
  const primaryLabel = showFullPrimary ? primary : shortHash(primary, 18, 12);
  return h('div', { class: 'detail-head' }, [
    h('div', [
      h('p', title),
      h('h2', { class: showFullPrimary ? 'full-address' : '', title: primary }, primaryLabel),
    ]),
    h('div', { class: 'detail-head-actions' }, [
      h('span', { class: ['type-badge', tone] }, badge),
      h('button', { class: 'icon-button', type: 'button', title: 'Copy', onClick: () => emit('copy', primary) }, [h(Copy, { size: 17 })]),
    ]),
  ]);
}

function accountField(label, account, emit) {
  return h('div', { class: 'detail-field' }, [
    h('dt', label),
    h('dd', [
      account ? h('button', { class: 'link-button', type: 'button', onClick: () => emit('go-address', account) }, account) : h('span', '-'),
    ]),
  ]);
}

function metric(label, value) {
  return h('div', { class: 'metric-box' }, [
    h('strong', String(value)),
    h('small', label),
  ]);
}
</script>
