<script setup>
import { useStore } from '@/store/live'
const store = useStore()
console.log(store.display_membership)
</script>
<script>
export default {
  name: 'member-ship'
}
</script>

<template>
  <div class="membership">
    <img
      class="membership-image"
      :src="store.display_membership.img_src"
      loop="false"
      :class="{
        hidden: !store.display_membership.playing
      }"
    />
    <div
      class="membership-user"
      :class="{
        hidden: !store.display_membership.playing,
        'fade-in': !store.display_membership.fade_out,
        'fade-out': store.display_membership.fade_out
      }"
    >
      <div class="membership-face-container">
        <img class="membership-face" :src="store.display_membership.uface" />
      </div>
      <div class="membership-name-container">
        <span class="membership-name">{{ store.display_membership.uname }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.membership {
  position: absolute;
  width: 100vw;
  height: 100vh;
}

.membership-image {
  animation: fadeOut 1s ease 4s forwards;
  width: 100vw;
  height: 100vh;
  position: absolute;
}

.membership-user {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: end;
  z-index: 999;
  position: relative;
}

.membership-user.fade-in {
  animation: fadeInFromBottom 1s;
}
.membership-user.fade-out {
  animation: fadeOutToBottom 2s;
}

.membership-face-container {
  background-color: white;
  width: 200px;
  height: 200px;
  border: 2px solid rgb(110, 171, 211);
  border-radius: 90px;
  display: flex;
  justify-content: center;
  align-items: center;
}

.membership-face {
  width: 180px;
  height: 180px;
  border-radius: 90px;
}

.membership-name-container {
  background-color: white;
  width: 200px;
  height: 50px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid rgb(110, 171, 211);
  border-radius: 25px;
  margin-top: 10px;
  margin-bottom: 50px;
}

.membership-name {
  font-size: 32px;
}
</style>
