import {
  CloseOutlined,
  FilterOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import { Button, Dropdown, type MenuProps } from "antd";
import { useTranslation } from "react-i18next";

export type SearchFacet = {
  key: string;
  /** Text shown inside the facet pill, e.g. "Type: Asset". */
  label: string;
  onRemove: () => void;
};

export type SearchBarProps = {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  facets?: SearchFacet[];
  /** Items of the filter dropdown. Omit to hide the dropdown toggle. */
  filterItems?: MenuProps["items"];
  selectedFilterKeys?: string[];
  onFilterSelect?: (key: string) => void;
};

/**
 * Odoo-style search bar: facet pills and a text input in one box, with an
 * optional filter dropdown at the end.
 */
export function SearchBar({
  value,
  onChange,
  placeholder,
  facets = [],
  filterItems,
  selectedFilterKeys = [],
  onFilterSelect,
}: SearchBarProps) {
  const { t } = useTranslation();

  return (
    <div
      className="surface flex min-h-[28px] w-full min-w-0 max-w-[560px] items-center gap-1 px-2"
      style={{ borderRadius: "var(--omed-radius)" }}
    >
      <SearchOutlined style={{ color: "var(--omed-text-muted)" }} />
      {facets.map((facet) => (
        <span
          key={facet.key}
          className="inline-flex shrink-0 items-center gap-1 px-1.5 text-xs"
          style={{
            borderRadius: "var(--omed-radius-sm)",
            background: "var(--omed-tint)",
            color: "var(--omed-tint-text)",
          }}
        >
          {facet.label}
          <button
            type="button"
            aria-label={t("common.removeFilter", { name: facet.label })}
            onClick={facet.onRemove}
            className="inline-flex cursor-pointer border-0 bg-transparent p-0"
            style={{ color: "inherit" }}
          >
            <CloseOutlined style={{ fontSize: 9 }} />
          </button>
        </span>
      ))}
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder ?? t("common.search")}
        aria-label={placeholder ?? t("common.search")}
        className="min-w-[80px] flex-1 border-0 bg-transparent py-1 outline-none"
        style={{ color: "var(--omed-text)", font: "inherit" }}
      />
      {filterItems ? (
        <Dropdown
          trigger={["click"]}
          menu={{
            items: filterItems,
            selectable: true,
            multiple: true,
            selectedKeys: selectedFilterKeys,
            onClick: ({ key }) => onFilterSelect?.(key),
          }}
        >
          <Button
            type="text"
            size="small"
            icon={<FilterOutlined />}
            aria-label={t("common.filters")}
          />
        </Dropdown>
      ) : null}
    </div>
  );
}
