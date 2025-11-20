#!/usr/bin/env node

/**
 * JavaScript Runtime Helper for ngc-go
 * 
 * This helper provides JavaScript runtime capabilities for the Go compiler:
 * - Creating functions from source code (with Trusted Types support)
 * - Executing functions with arguments
 * 
 * Usage:
 *   node index.js new-function < JSON_INPUT
 *   node index.js execute < JSON_INPUT
 */

const readline = require('readline');
const fs = require('fs');
const os = require('os');
const path = require('path');

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
  terminal: false
});

let inputData = '';

// Use file-based cache for persistence across processes
const cacheDir = path.join(os.tmpdir(), 'ngc-go-js-runtime');
const cacheFile = path.join(cacheDir, 'functions.json');

// Ensure cache directory exists
if (!fs.existsSync(cacheDir)) {
  fs.mkdirSync(cacheDir, { recursive: true });
}

// Load cache from file
function loadCache() {
  if (fs.existsSync(cacheFile)) {
    try {
      const data = fs.readFileSync(cacheFile, 'utf8');
      return JSON.parse(data);
    } catch (e) {
      return {};
    }
  }
  return {};
}

// Save cache to file
function saveCache(cache) {
  fs.writeFileSync(cacheFile, JSON.stringify(cache, null, 2), 'utf8');
}

rl.on('line', (line) => {
  inputData += line + '\n';
});

rl.on('close', () => {
  try {
    const input = JSON.parse(inputData.trim());
    const command = process.argv[2];

    if (command === 'new-function') {
      handleNewFunction(input);
    } else if (command === 'execute') {
      handleExecute(input);
    } else {
      console.error(JSON.stringify({ error: `Unknown command: ${command}` }));
      process.exit(1);
    }
  } catch (error) {
    console.error(JSON.stringify({ error: error.message }));
    process.exit(1);
  }
});

/**
 * Handle new-function command
 * Input: { args: string[], body: string }
 * Output: { functionId: string, source: string }
 */
function handleNewFunction(input) {
  const { args, body } = input;
  
  if (!body) {
    throw new Error('function body is required');
  }

  // Create function source code
  const paramList = args ? args.join(', ') : '';
  const source = `(function anonymous(${paramList}) { ${body} })`;

  // Try to use Trusted Types if available
  let functionId;
  let trustedFunction;

  if (typeof globalThis !== 'undefined' && globalThis.trustedTypes) {
    try {
      const policy = globalThis.trustedTypes.createPolicy('angular#unsafe-jit', {
        createScript: (s) => s,
      });
      trustedFunction = policy.createScript(source);
      functionId = `trusted_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    } catch (e) {
      // Fall back to regular function if Trusted Types fails
      functionId = `fn_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
  } else {
    functionId = `fn_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  // Store function in file-based cache for persistence across processes
  const cache = loadCache();
  cache[functionId] = {
    source,
    trustedFunction: trustedFunction ? trustedFunction.toString() : null
  };
  saveCache(cache);

  console.log(JSON.stringify({
    functionId,
    source,
  }));
}

/**
 * Handle execute command
 * Input: { functionId: string, args: any[] } or { source: string, args: any[] }
 * Output: { result: any } or { error: string }
 */
function handleExecute(input) {
  const { functionId, args, source: directSource } = input;

  let source = directSource;
  let trustedFunction = null;

  // Priority: functionId from cache > direct source
  if (functionId) {
    // Try to load from cache first
    const cache = loadCache();
    if (cache[functionId]) {
      const funcData = cache[functionId];
      source = funcData.source;
      trustedFunction = funcData.trustedFunction;
    } else if (directSource) {
      // Fallback to direct source if functionId not found in cache
      source = directSource;
    } else {
      throw new Error(`Function ${functionId} not found in cache and no source provided`);
    }
  } else if (directSource) {
    // Use direct source if no functionId provided
    source = directSource;
  } else {
    throw new Error('Either functionId or source must be provided');
  }

  try {
    // Create and execute function
    let fn;
    if (trustedFunction && typeof globalThis !== 'undefined' && globalThis.trustedTypes) {
      // Use Trusted Types if available
      try {
        const policy = globalThis.trustedTypes.createPolicy('angular#unsafe-jit', {
          createScript: (s) => s,
        });
        const trustedScript = policy.createScript(trustedFunction);
        fn = eval(trustedScript.toString());
      } catch (e) {
        // Fall back to regular eval
        fn = eval(source);
      }
    } else {
      // Use regular eval
      fn = eval(source);
    }

    // Bind to globalThis to mimic new Function() behavior
    fn = fn.bind(globalThis);

    // Execute with arguments
    const result = fn(...(args || []));

    console.log(JSON.stringify({
      result,
    }));
  } catch (error) {
    console.error(JSON.stringify({
      error: error.message,
      stack: error.stack,
    }));
    process.exit(1);
  }
}

