import { auth } from "@omed/better-auth";

let _schema: ReturnType<typeof auth.api.generateOpenAPISchema>;
const getSchema = async () => (_schema ??= auth.api.generateOpenAPISchema());
export const OpenAPI = {
  getPaths: (prefix = "") =>
    getSchema().then(({ paths }) => {
      const reference: typeof paths = Object.create(null);
      for (const path of Object.keys(paths)) {
        const pathItem = paths[path];
        if (!pathItem) continue;
        const key = prefix + path;
        reference[key] = pathItem;
        for (const method of Object.keys(pathItem)) {
          const operation = (reference[key] as any)[method];
          if (operation) {
            operation.tags = ["Better Auth"];
          }
        }
      }
      return reference;
    }) as Promise<unknown>,
  components: getSchema().then(
    ({ components }) => components,
  ) as Promise<unknown>,
} as const;
