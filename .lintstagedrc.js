'use strict';
const path = require('path');

// Touching these packages requires rebuilding and testing every provider,
// since all three import them. See e.g. provider/compose/types.go.
const SHARED = ['provider/compose', 'provider/common'];

const PROVIDERS = [
  { dir: 'provider/defangaws',   pkg: './provider/defangaws/...'   },
  { dir: 'provider/defangazure', pkg: './provider/defangazure/...' },
  { dir: 'provider/defanggcp',   pkg: './provider/defanggcp/...'   },
];

/**
 * @param {string[]} files Absolute paths of staged .go files
 * @returns {string[]} Shell commands to run
 */
function goHook(files) {
  const rel = files.map(f => path.relative(process.cwd(), f));

  const touchesShared = rel.some(f => SHARED.some(s => f.startsWith(s + '/')));

  // Always test the shared compose package; add provider packages selectively.
  const testPkgs = ['./provider/compose/...'];
  const buildPkgs = [];
  for (const { dir, pkg } of PROVIDERS) {
    if (touchesShared || rel.some(f => f.startsWith(dir + '/'))) {
      testPkgs.push(pkg);
      buildPkgs.push(pkg);
    }
  }

  const cmds = [
    // Lint all provider + test code regardless of which files changed;
    // golangci-lint is fast in cache-hit mode.
    'golangci-lint run --fix --timeout 5m ./provider/... ./tests/...',
  ];
  if (buildPkgs.length > 0) {
    cmds.push(`go build ${buildPkgs.join(' ')}`);
  }
  cmds.push(`go test -count=1 -timeout 5m ${testPkgs.join(' ')}`);
  return cmds;
}

module.exports = { '**/*.go': goHook };
