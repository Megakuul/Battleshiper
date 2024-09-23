import { Adapter } from '@sveltejs/kit';

export interface AdapterOptions {
	debug?: bool;
}

export default function plugin(options?: AdapterOptions): Adapter;