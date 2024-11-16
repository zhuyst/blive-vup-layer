<script setup>
import { ref, watch, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import { useStore } from '@/store/live'

const store = useStore()

const sc_list_ref = ref(null)
const { sc_list } = storeToRefs(store)

watch(
  () => sc_list.value,
  async () => {
    await nextTick()
    sc_list_ref.value.scrollTo({
      top: sc_list_ref.value.scrollHeight,
      behavior: 'smooth'
    })
  },
  { deep: true }
)
</script>
<script>
export default {
  name: 'sc-list'
}
</script>
<template>
  <div class="sc-list-container" ref="sc_list_ref">
    <div class="sc-list">
      <div
        class="sc-item"
        v-for="item in store.sc_list"
        :key="item.msg_id"
        :class="{
          'fade-in': !item.fade_out,
          'fade-out': item.fade_out
        }"
      >
        <div class="sc-content">{{ item.msg }}</div>
        <div class="sc-triangle"></div>
        <div class="sc-user">
          <img class="sc-face" :src="item.uface" />
          <div class="sc-name">{{ item.uname }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style sroped>
.sc-list-container {
  width: 100%;
  height: 50%;
  overflow-y: auto;
}

.sc-list {
}

.sc-list:last-child {
  margin-bottom: 0;
}

.sc-item {
  width: 100%;
  margin-bottom: 20px;

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

.sc-item.fade-in {
  animation: fadeInFromLeftPlus 1s;
}
.sc-item.fade-out {
  animation: fadeOutToRight 1s;
}

.sc-content {
  background-color: white;
  border-radius: 10px;
  padding: 10px;
  box-shadow: 5px 5px 10px #888;
  min-width: 200px;
  min-height: 34px;
  border: 1px solid rgb(110, 171, 211);
  font-size: 24px;
}

.sc-triangle {
  width: 0;
  height: 0;
  border: 12px solid;
  border-color: rgb(110, 171, 211) transparent transparent transparent;

  position: relative;
  left: 20px;
}

.sc-user {
  display: flex;
  align-items: center;
  margin-left: 15px;

  position: relative;
  bottom: 6px;
}

.sc-face {
  width: 32px;
  height: 32px;
  border-radius: 24px;
  margin-right: 5px;
}

.sc-name {
  font-size: 20px;
}
</style>
