#!/usr/bin/env node
// Receives release notes on stdin (written by semantic-release's
// generateNotesCmd contract) and writes them to /tmp/release-notes.md
// so GoReleaser can consume them via --release-notes.
//
// Using stdin avoids passing release note content through shell expansion,
// which would be a shell-injection risk if commit messages contain $(), backticks, etc.

const fs = require("fs");

let notes = "";
process.stdin.setEncoding("utf8");
process.stdin.on("data", (chunk) => {
  notes += chunk;
});
process.stdin.on("end", () => {
  fs.writeFileSync("/tmp/release-notes.md", notes);
  process.stdout.write(notes); // pass through for semantic-release to use as the release body
});
