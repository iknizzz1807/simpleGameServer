// currentUser.ts
import { writable } from "svelte/store";

export interface User {
  id: string;
  username: string;
}

const initialUser: User | null = null;

export const currentUser = writable<User | null>(initialUser);

export const setUser = (user: User) => {
  currentUser.set(user);
};

export const clearUser = () => {
  currentUser.set(null);
};
