import assert from "node:assert/strict";
import test from "node:test";

import { appVersionLabel, updatePanelClass } from "../src/updatePanel.ts";

const baseStatus = {
  currentVersion: "v0.11.0",
  latestVersion: "",
  available: false,
  state: "idle",
  message: "",
  releaseURL: "",
  downloadedPath: "",
  logPath: "",
};

test("marks the update panel current when the app is already latest", () => {
  assert.equal(updatePanelClass({ ...baseStatus, state: "current" }), "update-panel current");
});

test("marks the update panel available when a new version exists", () => {
  assert.equal(
    updatePanelClass({ ...baseStatus, available: true, state: "available" }),
    "update-panel available",
  );
});

test("keeps the update panel available while an update is in progress", () => {
  for (const state of ["downloading", "downloaded", "installing"]) {
    assert.equal(updatePanelClass({ ...baseStatus, available: true, state }), "update-panel available");
  }
});

test("does not color the update panel before update status is known", () => {
  assert.equal(updatePanelClass(baseStatus), "update-panel");
});

test("uses a non-release fallback when the app version has not loaded", () => {
  assert.equal(appVersionLabel(""), "dev");
  assert.equal(appVersionLabel("  v0.11.0  "), "v0.11.0");
});
