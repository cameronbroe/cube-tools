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
        // For double-faced cards, also index by the front-face name so that
        // lookups using either the full "Front // Back" form or just "Front"
        // will resolve correctly (CubeCobra often stores only the front face).
        const slashIdx = card.name.indexOf(' // ');
        if (slashIdx !== -1) {
          priceMap.set(card.name.slice(0, slashIdx).toLowerCase(), price);
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
