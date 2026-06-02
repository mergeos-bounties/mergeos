#!/usr/bin/env node

const { main } = require("../src/cli");

main(process.argv.slice(2)).catch((error) => {
  const message = error && error.message ? error.message : String(error);
  console.error(`MergeIDE error: ${message}`);
  process.exitCode = 1;
});
