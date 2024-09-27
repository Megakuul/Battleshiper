import { writable } from "svelte/store";

/** @type {import("svelte/store").Writable<import("$lib/adapter/user/fetchinfo").fetchInfoOutput|undefined>} */
export const UserInfo = writable(undefined); 

/** @type {import("svelte/store").Writable<import("$lib/adapter/resource/listrepository").listRepositoryOutput|undefined>} */
export const RepositoryInfo = writable(undefined);

/** @type {import("svelte/store").Writable<import("$lib/adapter/resource/listproject").listProjectOutput|undefined>} */
export const ProjectInfo = writable(undefined); 