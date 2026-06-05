"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const workflowPath = path.resolve(__dirname, "../../.github/workflows/mergeide-windows-exe.yml");

test("Windows exe workflow publishes the stable MergeIDE release asset contract", () => {
  const workflow = fs.readFileSync(workflowPath, "utf8");

  assert.match(workflow, /permissions:\s*\n\s*contents: write/);
  assert.match(workflow, /runs-on: windows-latest/);
  assert.match(workflow, /MERGEIDE_DEFAULT_RELEASE_TAG: mergeide-windows-latest/);
  assert.match(workflow, /MERGEIDE_EXE_NAME: MergeIDE-Windows-x64\.exe/);
  assert.match(workflow, /MERGEIDE_CHECKSUM_NAME: MergeIDE-Windows-x64\.exe\.sha256/);
  assert.match(workflow, /MERGEIDE_BUILD_INFO_NAME: MergeIDE-Windows-x64\.build\.json/);
  assert.match(workflow, /run: npm run build:exe/);
  assert.match(workflow, /run: \.\\dist\\MergeIDE-Windows-x64\.exe --help/);
  assert.match(workflow, /MERGEIDE_DOWNLOAD_URL=\$repoUrl\/releases\/download\/\$tag\/\$env:MERGEIDE_EXE_NAME/);
  assert.match(workflow, /MERGEIDE_CHECKSUM_URL=\$repoUrl\/releases\/download\/\$tag\/\$env:MERGEIDE_CHECKSUM_NAME/);
  assert.match(workflow, /MERGEIDE_BUILD_INFO_URL=\$repoUrl\/releases\/download\/\$tag\/\$env:MERGEIDE_BUILD_INFO_NAME/);
  assert.match(workflow, /uses: actions\/upload-artifact@v7/);
  assert.match(workflow, /name: MergeIDE-Windows-x64/);
  assert.match(workflow, /MergeIDE\/dist\/\$\{\{ env\.MERGEIDE_EXE_NAME \}\}/);
  assert.match(workflow, /MergeIDE\/dist\/\$\{\{ env\.MERGEIDE_CHECKSUM_NAME \}\}/);
  assert.match(workflow, /MergeIDE\/dist\/\$\{\{ env\.MERGEIDE_BUILD_INFO_NAME \}\}/);
  assert.match(workflow, /gh release upload \$tag/);
  assert.match(workflow, /--clobber/);
  assert.match(workflow, /gh release view \$tag --json assets,tagName,url/);
  assert.match(workflow, /missing required MergeIDE asset/);
  assert.match(workflow, /- Download: \$env:MERGEIDE_DOWNLOAD_URL/);
  assert.match(workflow, /- SHA256 file: \$env:MERGEIDE_CHECKSUM_URL/);
  assert.match(workflow, /- Build metadata: \$env:MERGEIDE_BUILD_INFO_URL/);
});
