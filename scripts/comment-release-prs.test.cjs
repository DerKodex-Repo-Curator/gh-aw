"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");

const source = fs.readFileSync(path.join(__dirname, "../.github/workflows/release.md"), "utf8");
const job = source.slice(source.indexOf("  comment_release_prs:"), source.indexOf("\nsteps:"));
const match = job.match(/^ +script: \|\n((?: {12}.*\n|\n)+)/m);
assert.ok(match, "Missing release PR commenting script");
const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor;
const script = new AsyncFunction("github", "context", "core", "process", match[1].replace(/^ {12}/gm, ""));

const previousPublishedAt = "2026-10-14T04:49:11Z";
const currentPublishedAt = "2026-10-14T19:18:56Z";
const marker = "<!-- gh-aw-release:v0.91.13 -->";

function pr(number, mergedAt, updatedAt = mergedAt || currentPublishedAt) {
  return { number, merged_at: mergedAt, updated_at: updatedAt, title: `PR ${number}`, html_url: `https://github.com/example/repository/pull/${number}` };
}

function harness(pages, { previousRelease = true, alreadyCommented = [] } = {}) {
  const created = [];
  let pagesRead = 0;
  const listPulls = () => {};
  const listComments = () => {};
  const currentRelease = { id: 13, tag_name: "v0.91.13", published_at: currentPublishedAt, html_url: "https://github.com/example/repository/releases/tag/v0.91.13" };
  const paginate = async (method, parameters) => {
    assert.equal(method, listComments);
    return alreadyCommented.includes(parameters.issue_number) ? [{ body: marker }] : [];
  };
  paginate.iterator = async function* (method, parameters) {
    assert.equal(method, listPulls);
    assert.deepEqual(parameters, { owner: "example", repo: "repository", state: "closed", sort: "updated", direction: "desc", per_page: 100 });
    for (const data of pages) {
      pagesRead += 1;
      yield { data };
    }
  };
  const github = {
    rest: {
      repos: {
        getRelease: async () => ({ data: currentRelease }),
        listReleases: async () => ({ data: previousRelease ? [currentRelease, { id: 12, tag_name: "v0.91.12", published_at: previousPublishedAt }] : [currentRelease] }),
      },
      pulls: { list: listPulls },
      issues: {
        listComments,
        createComment: async parameters => created.push(parameters),
      },
    },
    paginate,
  };
  const run = () =>
    script(
      github,
      { repo: { owner: "example", repo: "repository" } },
      {
        info: () => {},
        warning: () => {},
        setFailed: message => assert.fail(message),
      },
      { env: { RELEASE_TAG: "v0.91.13", RELEASE_ID: "13" } }
    );
  return { run, created, pagesRead: () => pagesRead };
}

test("lists all relevant pages and filters by merge time, not update time", async () => {
  const fixture = harness([[pr(1, null), pr(2, "2026-10-13T12:00:00Z", currentPublishedAt), pr(3, "2026-10-14T20:00:00Z")], [pr(4, "2026-10-14T12:00:00Z"), pr(5, "2026-10-14T04:00:00Z")], [pr(6, "2026-10-13T12:00:00Z")]]);
  await fixture.run();
  assert.equal(fixture.pagesRead(), 2);
  assert.deepEqual(
    fixture.created.map(comment => comment.issue_number),
    [4]
  );
  assert.ok(fixture.created[0].body.includes(marker));
});

for (const boundary of [previousPublishedAt, currentPublishedAt]) {
  test(`includes merge at release boundary ${boundary}`, async () => {
    const fixture = harness([[pr(1, boundary)]]);
    await fixture.run();
    assert.deepEqual(
      fixture.created.map(comment => comment.issue_number),
      [1]
    );
  });
}

test("does not stop at an old merge when that PR was updated recently", async () => {
  const fixture = harness([[pr(1, "2026-10-13T12:00:00Z", currentPublishedAt)], [pr(2, "2026-10-14T12:00:00Z")]]);
  await fixture.run();
  assert.equal(fixture.pagesRead(), 2);
  assert.deepEqual(
    fixture.created.map(comment => comment.issue_number),
    [2]
  );
});

test("skips duplicate release comments", async () => {
  const fixture = harness([[pr(1, "2026-10-14T12:00:00Z")]], { alreadyCommented: [1] });
  await fixture.run();
  assert.deepEqual(fixture.created, []);
});

test("handles an empty PR list", async () => {
  const fixture = harness([[]]);
  await fixture.run();
  assert.deepEqual(fixture.created, []);
});

test("does not list PRs when no previous release exists", async () => {
  const fixture = harness([[pr(1, "2026-10-14T12:00:00Z")]], { previousRelease: false });
  await fixture.run();
  assert.equal(fixture.pagesRead(), 0);
  assert.deepEqual(fixture.created, []);
});
