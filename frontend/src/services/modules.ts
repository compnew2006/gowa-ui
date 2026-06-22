import { api } from "@/services/api";

export interface ManagedModule {
  key: string;
  display_name: string;
  version: string;
  schema_version: number;
  dependencies?: string[];
  default_enabled: boolean;
  technical: boolean;
  global_enabled: boolean;
  organization_enabled: boolean;
  effective_enabled: boolean;
  installed_schema_version: number;
}

function moduleKeyPath(key: string): string {
  return encodeURIComponent(key);
}

export const modulesService = {
  listEffective: () => api.get<ManagedModule[]>("/modules/effective"),
  listGlobal: () => api.get<ManagedModule[]>("/admin/modules"),
  updateGlobal: (key: string, enabled: boolean) =>
    api.put(`/admin/modules/${moduleKeyPath(key)}`, { enabled }),
  listOrganization: (organizationId: string) =>
    api.get<ManagedModule[]>(`/organizations/${organizationId}/modules`),
  updateOrganization: (
    organizationId: string,
    key: string,
    enabled: boolean,
  ) =>
    api.put(
      `/organizations/${organizationId}/modules/${moduleKeyPath(key)}`,
      { enabled },
    ),
};
