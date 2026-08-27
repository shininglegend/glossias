export const SITE_NAME = "Glossias";

/** Build a browser tab title of the form `Page Name | Glossias`. */
export function pageTitle(page?: string): string {
  return page ? `${page} | ${SITE_NAME}` : SITE_NAME;
}

/** React Router `meta` export value for a page. */
export function pageMeta(page: string, description?: string) {
  return description
    ? [
        { title: pageTitle(page) },
        { name: "description", content: description },
      ]
    : [{ title: pageTitle(page) }];
}
