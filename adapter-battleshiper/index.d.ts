import { Adapter } from '@sveltejs/kit';

export interface AdapterOptions {
	debug?: boolean;
}

export default function plugin(options?: AdapterOptions): Adapter;