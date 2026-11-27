<template>
  <div>
    <h1>Nodes Management</h1>
    <a-button @click="showAddNodeModal = true">Add Node</a-button>
    <a-table :data-source="nodes" :columns="columns">
      <template #action="{ record }">
        <a-button @click="editNode(record)">Edit</a-button>
        <a-button type="danger" @click="deleteNode(record.id)">Delete</a-button>
      </template>
    </a-table>
    <a-modal v-model:visible="showAddNodeModal" title="Add Node">
      <!-- Add node form -->
    </a-modal>
  </div>
</template>

<script>
import { defineComponent, ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { message } from 'ant-design-vue';

export default defineComponent({
  setup() {
    const nodes = ref([]);
    const showAddNodeModal = ref(false);
    const router = useRouter();

    onMounted(async () => {
      const response = await fetch('/panel/api/nodes');
      const data = await response.json();
      if (data.success) {
        nodes.value = data.data;
      } else {
        message.error('Failed to load nodes');
      }
    });

    const columns = [
      { title: 'Name', dataIndex: 'name', key: 'name' },
      { title: 'Address', dataIndex: 'address', key: 'address' },
      { title: 'Status', dataIndex: 'status', key: 'status' },
      { title: 'Action', key: 'action', slots: { customRender: 'action' } },
    ];

    return {
      nodes, showAddNodeModal, columns,
      editNode: (node) => router.push(`/nodes/edit/${node.id}`),
      deleteNode: async (id) => {
        const response = await fetch(`/panel/api/nodes/${id}`, { method: 'DELETE' });
        const data = await response.json();
        if (data.success) {
          nodes.value = nodes.value.filter(node => node.id !== id);
          message.success('Node deleted');
        } else {
          message.error('Deletion failed');
        }
      },
    };
  },
});
</script>
