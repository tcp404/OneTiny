import type { UpdateStatusDTO } from "./types";

const availableStates = new Set(["available", "downloading", "downloaded", "installing"]);

export function updatePanelClass(status: Pick<UpdateStatusDTO, "available" | "state">): string {
  const classes = ["update-panel"];
  if (status.state === "current") {
    classes.push("current");
  } else if (status.available || availableStates.has(status.state)) {
    classes.push("available");
  }
  return classes.join(" ");
}

export function appVersionLabel(currentVersion: string): string {
  return currentVersion.trim() || "dev";
}
