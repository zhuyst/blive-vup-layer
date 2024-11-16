<script setup>
import { ref, watch, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import { useStore } from '@/store/live'

const store = useStore()

const danmu_list_ref = ref(null)
const { danmu_list } = storeToRefs(store)

watch(
  () => danmu_list.value,
  async () => {
    await nextTick()
    danmu_list_ref.value.scrollTo({
      top: danmu_list_ref.value.scrollHeight,
      behavior: 'smooth'
    })
  },
  { deep: true }
)
</script>
<script>
export default {
  name: 'danmu-list'
}
</script>

<template>
  <div class="danmu-list-container" ref="danmu_list_ref">
    <div class="danmu-list">
      <div class="danmu-item" v-for="item in store.danmu_list" :key="item.msg_id">
        <div class="danmu-user">
          <img class="danmu-face" :src="item.uface" />
          <div class="danmu-name">{{ item.uname }}</div>
          <div class="danmu-medal" v-if="item.fans_medal_wearing_status">
            <div class="danmu-medal-name">{{ item.fans_medal_name }}</div>
            <div class="danmu-medal-level">{{ item.fans_medal_level }}</div>
          </div>
          <div>：</div>
        </div>
        <div class="danmu-msg">
          <img v-if="item.dm_type === 1" class="danmu-emoji" :src="item.emoji_img_url" />
          <template v-else>
            <template v-for="(t, j) in item.rich_text" :key="j">
              <img v-if="t.type === 'image'" :src="t.img_url" class="danmu-normal-emoji" />
              <span v-else>{{ t.text }}</span>
            </template>
          </template>
        </div>
        <div class="danmu-line" v-if="index != store.danmu_list.length - 1"></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.danmu-list-container {
  width: 100%;
  height: 80%;
  border: 2px solid rgb(110, 171, 211);
  overflow-y: auto;
  scroll-behavior: smooth;
}

.danmu-item {
  animation: fadeInFromLeft 0.5s;
  padding: 5px;
}

.danmu-user {
  display: flex;
  align-items: center;
  font-size: 14px;
}

.danmu-face {
  width: 24px;
  height: 24px;
  border-radius: 24px;
  margin-right: 5px;
}

.danmu-name {
  margin-right: 5px;
}

.danmu-medal {
  display: flex;
  justify-content: center;
  align-items: center;
  border: 0.5px solid rgb(103, 232, 255);
  font-size: 12px;
  margin-right: 5px;
}

.danmu-medal-name {
  padding: 2px 5px;
  color: rgb(255, 255, 255);
  background-image: linear-gradient(90deg, rgb(45, 8, 85), rgb(157, 155, 255));
}

.danmu-medal-level {
  padding: 2px;
  color: rgb(45, 8, 85);
}

.danmu-emoji {
  width: 48px;
  height: 48px;
}

.danmu-msg {
  margin-top: 5px;
  font-size: 16px;
}

.danmu-normal-emoji {
  width: 16px;
  height: 16px;
}

.danmu-line {
  width: 100%;
  height: 1px;
  background-color: rgb(110, 171, 211);
  margin-top: 5px;
}
</style>
