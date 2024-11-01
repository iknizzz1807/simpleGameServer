// store.ts
import { writable } from "svelte/store";

export const messages = writable<string[]>([]);
export const numOfPlayers = writable<number>(0);
