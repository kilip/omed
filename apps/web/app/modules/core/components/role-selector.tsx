import { Radio, Tag } from "antd";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Panel } from "~/components/Panel";
import { moduleLabel, type Role, roleLabel } from "~/lib/roles";

type RoleSelectorProps = {
  roles: Role[];
  value?: string[];
  onChange?: (value: string[]) => void;
  disabled?: boolean;
};

export function RoleSelector({
  roles,
  value = [],
  onChange,
  disabled = false,
}: RoleSelectorProps) {
  const { t } = useTranslation();

  const getModule = (roleId: string) => {
    const role = roles.find((r) => r.id === roleId);
    return role?.module || roleId.split(":")[0] || "core";
  };

  const grouped = useMemo(() => {
    const map = new Map<string, Role[]>();
    for (const r of roles) {
      if (!r.id) continue;
      const mod = r.module || r.id.split(":")[0] || "core";
      const list = map.get(mod) ?? [];
      list.push(r);
      map.set(mod, list);
    }
    return Array.from(map.entries()).map(([mod, items]) => ({
      moduleId: mod,
      label: moduleLabel(t, mod),
      roles: items,
    }));
  }, [roles, t]);

  const select = (id: string) => {
    if (disabled || value.includes(id)) return;
    const mod = getModule(id);
    const withoutModule = value.filter((v) => getModule(v) !== mod);
    onChange?.([...withoutModule, id]);
  };

  return (
    <div className="flex flex-col gap-3">
      {grouped.map((group) => {
        const selectedCount = group.roles.filter(
          (r) => r.id && value.includes(r.id),
        ).length;

        return (
          <Panel
            key={group.moduleId}
            strong
            title={
              <div className="flex items-center gap-2">
                <span>{group.label}</span>
                {selectedCount > 0 ? (
                  <Tag color="blue" className="text-xs">
                    {selectedCount}
                  </Tag>
                ) : null}
              </div>
            }
          >
            <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
              {group.roles.map((r) => {
                if (!r.id) return null;
                const roleId = r.id;
                const checked = value.includes(roleId);
                const name = roleLabel(t, roleId, r.name);

                return (
                  <div
                    key={roleId}
                    className="flex items-start gap-2.5 p-2.5 rounded transition-all select-none"
                    style={{
                      borderRadius: "var(--omed-radius)",
                      border: checked
                        ? "1px solid var(--omed-primary)"
                        : "1px solid var(--omed-border-soft)",
                      background: checked ? "var(--omed-tint)" : "transparent",
                    }}
                  >
                    <Radio
                      checked={checked}
                      disabled={disabled}
                      className="w-full !flex !items-start"
                      onChange={() => select(roleId)}
                    >
                      <div className="min-w-0">
                        <div className="flex items-center gap-1.5 flex-wrap">
                          <span className="font-medium text-sm leading-tight">
                            {name}
                          </span>
                          <Tag className="text-xs m-0">{roleId}</Tag>
                        </div>
                        {r.description ? (
                          <div
                            className="text-xs mt-1 leading-snug font-normal"
                            style={{ color: "var(--omed-text-secondary)" }}
                          >
                            {r.description}
                          </div>
                        ) : null}
                      </div>
                    </Radio>
                  </div>
                );
              })}
            </div>
          </Panel>
        );
      })}
    </div>
  );
}
