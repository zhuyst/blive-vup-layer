<script setup>
import { ref, watch, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import { useStore } from '@/store/live'

const store = useStore()

const gift_list_ref = ref(null)
const { gift_list } = storeToRefs(store)

watch(
  () => gift_list.value,
  async () => {
    await nextTick()
    gift_list_ref.value.scrollTo({
      top: gift_list_ref.value.scrollHeight,
      behavior: 'smooth'
    })
  },
  { deep: true }
)
</script>
<script>
export default {
  name: 'gift-list'
}
</script>

<template>
  <div class="gift-list-container" ref="gift_list_ref">
    <div class="gift-list">
      <div class="gift-item" v-for="item in store.gift_list" :key="item.msg_id">
        <div class="gift-header">
          <img class="gift-face" :src="item.uface" />
          <div class="gift-right">
            <div class="gift-uname">{{ item.uname }}</div>
            <div class="gift-price">CN¥{{ item.rmb }}</div>
          </div>
        </div>
        <div class="gift-content">投喂 {{ item.gift_name }}x{{ item.gift_num }}</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.gift-list-container {
  width: 30%;
  margin: 3%;
  padding: 10px;

  overflow-y: auto;
}

.gift-list {
}

.gift-list:last-child {
  margin-bottom: 0;
}

.gift-item {
  color: white;
  font-size: 18px;
  border-radius: 12px;
  margin-bottom: 20px;

  animation: fadeInFromLeft 1s;

  text-shadow:
    -2px -2px #000000,
    -2px -1px #000000,
    -2px 0px #000000,
    -2px 1px #000000,
    -2px 2px #000000,
    -1px -2px #000000,
    -1px -1px #000000,
    -1px 0px #000000,
    -1px 1px #000000,
    -1px 2px #000000,
    0px -2px #000000,
    0px -1px #000000,
    0px 0px #000000,
    0px 1px #000000,
    0px 2px #000000,
    1px -2px #000000,
    1px -1px #000000,
    1px 0px #000000,
    1px 1px #000000,
    1px 2px #000000,
    2px -2px #000000,
    2px -1px #000000,
    2px 0px #000000,
    2px 1px #000000,
    2px 2px #000000;
}

.gift-header {
  display: flex;
  align-items: center;
  padding: 10px 25px;
  background-color: rgb(221, 79, 0);
  border-radius: 10px 10px 0 0;
}

.gift-face {
  width: 32px;
  height: 32px;
  border-radius: 24px;
}

.gift-right {
  margin-left: 10px;
}

.gift-uname {
}

.gift-price {
}

.gift-content {
  padding: 10px 25px;
  background-color: rgb(236, 123, 0);
  border-radius: 0 0 10px 10px;
}
</style>
