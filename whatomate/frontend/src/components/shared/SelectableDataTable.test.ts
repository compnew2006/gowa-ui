// @vitest-environment happy-dom

import { describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";
import { defineComponent } from "vue";
import SelectableDataTable from "./SelectableDataTable.vue";

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: any) => {
      if (params) {
        return `${key}:${JSON.stringify(params)}`
      }
      return key
    },
    locale: { value: 'en' }
  }),
}));

const CheckboxStub = defineComponent({
  props: ["checked"],
  emits: ["update:checked"],
  template: `<input type="checkbox" :checked="checked" @change="$emit('update:checked', $event.target.checked)" />`,
});

const ButtonStub = defineComponent({
  emits: ["click"],
  template: `<button @click="$emit('click', $event)"><slot /></button>`,
});

const passthroughStub = defineComponent({
  template: `<slot />`,
});

describe("SelectableDataTable", () => {
  const columns = [
    { key: "phone", label: "Phone Number" },
    { key: "name", label: "Name" },
  ];

  const items = [
    { id: "1", phone: "12345", name: "Alice" },
    { id: "2", phone: "67890", name: "Bob" },
  ];

  it("renders headers and items data correctly", () => {
    const wrapper = mount(SelectableDataTable, {
      props: {
        items,
        columns,
        currentPage: 1,
        totalPages: 1,
        totalItems: 2,
        pageSize: 25,
        selectedIds: new Set<string | number>(),
        isAllMatchingSelected: false,
        isAllPageSelected: false,
        selectedCount: 0,
      },
      global: {
        stubs: {
          Checkbox: CheckboxStub,
          Button: ButtonStub,
          Select: passthroughStub,
          SelectTrigger: passthroughStub,
          SelectValue: passthroughStub,
          SelectContent: passthroughStub,
          SelectItem: passthroughStub,
          Skeleton: passthroughStub,
        },
      },
    });

    // Check header labels are rendered
    expect(wrapper.text()).toContain("Phone Number");
    expect(wrapper.text()).toContain("Name");

    // Check rows data are rendered
    expect(wrapper.text()).toContain("12345");
    expect(wrapper.text()).toContain("Alice");
    expect(wrapper.text()).toContain("67890");
    expect(wrapper.text()).toContain("Bob");
  });

  it("emits toggle-row event when row checkbox is clicked", async () => {
    const wrapper = mount(SelectableDataTable, {
      props: {
        items,
        columns,
        currentPage: 1,
        totalPages: 1,
        totalItems: 2,
        pageSize: 25,
        selectedIds: new Set<string | number>(),
        isAllMatchingSelected: false,
        isAllPageSelected: false,
        selectedCount: 0,
      },
      global: {
        stubs: {
          Checkbox: CheckboxStub,
          Button: ButtonStub,
          Select: passthroughStub,
          SelectTrigger: passthroughStub,
          SelectValue: passthroughStub,
          SelectContent: passthroughStub,
          SelectItem: passthroughStub,
          Skeleton: passthroughStub,
        },
      },
    });

    const checkboxes = wrapper.findAll('input[type="checkbox"]');
    // First checkbox is header select-all, second is first row checkbox
    expect(checkboxes.length).toBe(3);

    await checkboxes[1].setValue(true);
    expect(wrapper.emitted("toggle-row")).toBeTruthy();
    expect(wrapper.emitted("toggle-row")![0][0]).toEqual(items[0]);
  });

  it("emits toggle-page event when header select-all checkbox is clicked", async () => {
    const wrapper = mount(SelectableDataTable, {
      props: {
        items,
        columns,
        currentPage: 1,
        totalPages: 1,
        totalItems: 2,
        pageSize: 25,
        selectedIds: new Set<string | number>(),
        isAllMatchingSelected: false,
        isAllPageSelected: false,
        selectedCount: 0,
      },
      global: {
        stubs: {
          Checkbox: CheckboxStub,
          Button: ButtonStub,
          Select: passthroughStub,
          SelectTrigger: passthroughStub,
          SelectValue: passthroughStub,
          SelectContent: passthroughStub,
          SelectItem: passthroughStub,
          Skeleton: passthroughStub,
        },
      },
    });

    const headerCheckbox = wrapper.find('input[type="checkbox"]');
    await headerCheckbox.setValue(true);
    expect(wrapper.emitted("toggle-page")).toBeTruthy();
  });

  it("displays running count and emits select-all-matching event when clicked", async () => {
    const wrapper = mount(SelectableDataTable, {
      props: {
        items,
        columns,
        currentPage: 1,
        totalPages: 2,
        totalItems: 50,
        pageSize: 2,
        selectedIds: new Set<string | number>(["1", "2"]),
        isAllMatchingSelected: false,
        isAllPageSelected: true,
        selectedCount: 2,
      },
      global: {
        stubs: {
          Checkbox: CheckboxStub,
          Button: ButtonStub,
          Select: passthroughStub,
          SelectTrigger: passthroughStub,
          SelectValue: passthroughStub,
          SelectContent: passthroughStub,
          SelectItem: passthroughStub,
          Skeleton: passthroughStub,
        },
      },
    });

    // Check count is displayed via key
    expect(wrapper.text()).toContain("selectableTable.selectedCount");

    // Click select all matching
    const selectAllMatchingBtn = wrapper.find("button");
    expect(selectAllMatchingBtn.text()).toContain("selectableTable.selectAllMatching");
    await selectAllMatchingBtn.trigger("click");

    expect(wrapper.emitted("select-all-matching")).toBeTruthy();
  });
});
