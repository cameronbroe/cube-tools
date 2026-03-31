import * as core from '@actions/core';

const BATCH_SIZE = 75;
const SCRYFALL_COLLECTION_URL = 'https://api.scryfall.com/cards/collection';
const USER_AGENT = 'cubecobra-price-action/1.0';

/**
 * Parse the cube ID from a CubeCobra URL or a bare ID.
 * Accepted URL patterns:
 *   https://cubecobra.com/cube/overview/<id>
 *   https://cubecobra.com/cube/list/<id>
 *   https://cubecobra.com/cube/<anything>/<id>
 *   Just a bare cube ID with no domain
 */
function parseCubeId(input) {
  const trimmed = input.trim().replace(/\/$/, '');
  const match = trimmed.match(/cubecobra\.com\/cube\/[^/]+\/([^/?#]+)/);
  if (match) {
    return match[1];
  }
  // Treat as a bare cube ID; reject if it looks like a URL we couldn't parse
  if (trimmed.includes('/') || trimmed.includes('.')) {
    throw new Error(`Could not parse a cube ID from: ${input}`);
  }
  return trimmed;
}

/**
 * Extract the card name from a CubeCobra card entry.
 * Handles: plain string, { name }, { details: { name } }
 */
function extractName(entry) {
  if (typeof entry === 'string') return entry;
  if (entry && typeof entry === 'object') {
    if (typeof entry.name === 'string' && entry.name) return entry.name;
    if (entry.details && typeof entry.details.name === 'string' && entry.details.name) {
      return entry.details.name;
    }
  }
  return null;
}

/**
 * Return the lowest non-null paper price (usd, usd_foil, usd_etched) for a card,
 * or null if none are available.
 */
function lowestPaperPrice(prices) {
  const candidates = [prices.usd, prices.usd_foil, prices.usd_etched]
    .filter((p) => p != null)
    .map(Number);
  return candidates.length > 0 ? Math.min(...candidates) : null;
}

/** Sleep for the given number of milliseconds. */
function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Fallback resolution for a single card name that the collection endpoint
 * could not match.  Two strategies are attempted in order:
 *
 *   1. /cards/named?fuzzy=<name> — handles Room cards whose full "A // B"
 *      oracle name may not be matched by the collection endpoint, as well as
 *      minor name variations.
 *   2. /cards/search?q=flavor_name:"<name>" — handles Universes Beyond /
 *      Secret Lair cards (e.g. "Endwalker", "Vaan, Aspiring Sky Pirate") that
 *      are stored in CubeCobra by their displayed flavor name rather than the
 *      Scryfall oracle name.
 *
 * Returns the lowest paper price (number) on success, or null if the card
 * still cannot be resolved.
 */
async function resolveNotFound(name) {
  // Strategy 1: fuzzy named lookup
  try {
    const resp = await fetch(
      `https://api.scryfall.com/cards/named?fuzzy=${encodeURIComponent(name)}`,
      { headers: { 'User-Agent': USER_AGENT } },
    );
    if (resp.ok) {
      const card = await resp.json();
      return lowestPaperPrice(card.prices || {});
    }
  } catch (_) {
    // fall through to strategy 2
  }

  // Strategy 2: flavor-name search (Universes Beyond / Secret Lair alternate names)
  try {
    await sleep(110);
    const resp = await fetch(
      `https://api.scryfall.com/cards/search?q=${encodeURIComponent(`flavor_name:"${name}"`)}`,
      { headers: { 'User-Agent': USER_AGENT } },
    );
    if (resp.ok) {
      const data = await resp.json();
      if (data.data && data.data.length > 0) {
        return lowestPaperPrice(data.data[0].prices || {});
      }
    }
  } catch (_) {
    // nothing more to try
  }

  return null;
}

async function run() {
  const cubecobraLink = core.getInput('cubecobra_link', { required: true });

  // 1. Parse the cube ID
  let cubeId;
  try {
    cubeId = parseCubeId(cubecobraLink);
  } catch (err) {
    core.setFailed(err.message);
    return;
  }
  core.info(`Cube ID: ${cubeId}`);

  // 2. Fetch the cube JSON from the CubeCobra API
  const apiUrl = `https://cubecobra.com/cube/api/cubeJSON/${cubeId}`;
  core.info(`Fetching cube data from: ${apiUrl}`);

  let cubeData;
  try {
    const resp = await fetch(apiUrl, { headers: { 'User-Agent': USER_AGENT } });
    if (!resp.ok) {
      core.setFailed(`CubeCobra API returned HTTP ${resp.status} for cube ID '${cubeId}'.`);
      return;
    }
    cubeData = await resp.json();
  } catch (err) {
    core.setFailed(`Could not reach CubeCobra API: ${err.message}`);
    return;
  }

  // 3. Extract mainboard card names
  const mainboard =
    cubeData.mainboard ||
    (cubeData.cards && cubeData.cards.mainboard) ||
    [];

  if (!mainboard.length) {
    core.setFailed('No mainboard cards found in the cube data.');
    return;
  }

  const cardNames = [];
  for (const entry of mainboard) {
    const name = extractName(entry);
    if (name) {
      cardNames.push(name);
    } else {
      core.warning(`Could not determine card name for entry: ${JSON.stringify(entry)}`);
    }
  }
  core.info(`Found ${cardNames.length} mainboard cards.`);

  // 4. Batch-query Scryfall for prices (max 75 identifiers per request)
  // Maps lowercased card name -> lowest paper price (number).
  // Duplicate card names in a cube are fine: the price lookup is the same
  // for every copy, and the summing loop below counts each copy separately.
  const priceMap = new Map();
  const noPrice = [];    // found on Scryfall but all paper price fields are null
  const notFound = [];   // Scryfall could not match the card name at all

  for (let batchStart = 0; batchStart < cardNames.length; batchStart += BATCH_SIZE) {
    const batch = cardNames.slice(batchStart, batchStart + BATCH_SIZE);
    const identifiers = batch.map((name) => ({ name }));

    let scryfallData;
    try {
      const resp = await fetch(SCRYFALL_COLLECTION_URL, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'User-Agent': USER_AGENT,
        },
        body: JSON.stringify({ identifiers }),
      });
      if (!resp.ok) {
        const body = await resp.text();
        core.setFailed(`Scryfall API returned HTTP ${resp.status}: ${body}`);
        return;
      }
      scryfallData = await resp.json();
    } catch (err) {
      core.setFailed(`Could not reach Scryfall API: ${err.message}`);
      return;
    }

    for (const card of scryfallData.data || []) {
      const price = lowestPaperPrice(card.prices || {});
      if (price !== null) {
        priceMap.set(card.name.toLowerCase(), price);
        // For double-faced / split / adventure cards, also index by each
        // individual face name so that lookups using either the full
        // "Front // Back" form, just "Front", or just "Back" all resolve
        // correctly (CubeCobra may store only the front face or the back face).
        const slashIdx = card.name.indexOf(' // ');
        if (slashIdx !== -1) {
          priceMap.set(card.name.slice(0, slashIdx).toLowerCase(), price);
          priceMap.set(card.name.slice(slashIdx + 4).toLowerCase(), price);
        }
        // Some cards have a flavor/alternate name (e.g. "Endwalker" for
        // "Brainstorm"). Scryfall returns the canonical name but CubeCobra may
        // store the flavor name, so index by it as well.
        if (card.flavor_name) {
          priceMap.set(card.flavor_name.toLowerCase(), price);
        }
      } else {
        noPrice.push(card.name);
      }
    }

    for (const nf of scryfallData.not_found || []) {
      notFound.push(nf.name || JSON.stringify(nf));
    }

    // Respect Scryfall's rate-limit (10 req/s recommended)
    if (batchStart + BATCH_SIZE < cardNames.length) {
      await sleep(110);
    }
  }

  // 4b. Fallback: individually resolve names the collection endpoint could not
  //     match (e.g. Room card full names, Universes Beyond flavor names).
  //     Deduplicate so we only attempt each unique name once.
  const attempted = new Set();
  const trulyNotFound = [];
  for (const nfName of notFound) {
    const lc = nfName.toLowerCase();
    if (priceMap.has(lc) || attempted.has(lc)) {
      continue;
    }
    attempted.add(lc);
    await sleep(110);
    const price = await resolveNotFound(nfName);
    if (price !== null) {
      priceMap.set(lc, price);
    } else {
      trulyNotFound.push(nfName);
    }
  }
  notFound.length = 0;
  notFound.push(...trulyNotFound);

  // 5. Sum up the total cost
  let total = 0;
  const unresolved = [];

  for (const name of cardNames) {
    const price = priceMap.get(name.toLowerCase());
    if (price != null) {
      total += price;
    } else {
      unresolved.push(name);
    }
  }

  if (notFound.length > 0) {
    core.info('\nCards not found on Scryfall (excluded from total):');
    for (const nf of notFound) core.info(`  - ${nf}`);
  }

  if (noPrice.length > 0) {
    core.info('\nCards found on Scryfall but with no paper price data (excluded from total):');
    for (const np of noPrice) core.info(`  - ${np}`);
  }

  if (unresolved.length > 0) {
    core.info('\nCards excluded from total (no price available):');
    for (const u of unresolved) core.info(`  - ${u}`);
  }

  const separator = '='.repeat(50);
  core.info(`\n${separator}`);
  core.info(`Cube:          ${cubeId}`);
  core.info(`Cards priced:  ${cardNames.length - unresolved.length} / ${cardNames.length}`);
  core.info(`Total cost:    $${total.toFixed(2)} USD`);
  core.info(separator);
}

run();
