import { currentUser } from "$lib/stores/currentUser";
import { redirect } from "@sveltejs/kit";
import { get } from "svelte/store";

export const load = async ({ url }) => {
  const user = get(currentUser);

  if (!user && url.pathname !== "/") {
    // If !homepage then redirect to avoid infinite redirect
    throw redirect(302, "/");
  }

  return { user };
};
