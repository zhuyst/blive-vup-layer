<script setup>
import { onMounted, reactive, nextTick, ref } from 'vue'
import imgSrc from '@/assets/captain_test_loop_1.webp'
import noFaceSrc from '@/assets/noface.gif'

let protocol = 'ws'
if (location.protocol === 'https:') {
  protocol = 'wss'
}
const serverUrl = protocol + '://' + location.host + '/server/ws'

function getUUID() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
    var r = (Math.random() * 16) | 0,
      v = c == 'x' ? r : (r & 0x3) | 0x8
    return v.toString(16)
  })
}

const state = reactive({
  is_connect_websocket: false,
  is_connect_room: false,
  connect_message: '正在连接至直播间',

  room_info: {
    room_id: 0,
    uname: '',
    uface: noFaceSrc
  },

  membership_list: [],
  display_membership: {
    uname: '',
    uface: noFaceSrc,

    img_src: '',
    playing: false,
    fade_out: false
  },

  danmu_list: [],
  sc_list: [],
  gift_list: []
})

const sendMemberShip = (data) => {
  if (!data) {
    data = {
      uname: '青云',
      uface: noFaceSrc
    }
  }
  state.membership_list.push(data)
  consumeMemberShipList()
}

const consumeMemberShipList = () => {
  if (state.display_membership.playing) {
    return
  }
  if (state.membership_list.length === 0) {
    return
  }

  const display = state.membership_list.splice(0, 1)[0]
  state.display_membership.uname = display.uname
  state.display_membership.uface = display.uface
  state.display_membership.playing = true
  state.display_membership.img_src = imgSrc

  setTimeout(() => {
    state.display_membership.fade_out = true

    setTimeout(() => {
      state.display_membership.img_src = ''
      state.display_membership.fade_out = false
      state.display_membership.playing = false
      setTimeout(consumeMemberShipList, 100)
    }, 2000)
  }, 4000)
}

const danmu_list = ref(null)
const sendDanmu = async (data) => {
  let msg_id
  if (!data) {
    msg_id = getUUID()
    data = {
      msg_id: msg_id,
      uname: '青云',
      uface: noFaceSrc,

      fans_medal_name: '巫女酱',
      fans_medal_level: 21,
      fans_medal_wearing_status: true,

      msg: '弹幕内容' + msg_id
    }
  } else {
    msg_id = data.msg_id
  }
  state.danmu_list.push(data)
  if (state.danmu_list.length >= 50) {
    state.danmu_list.splice(0, 1)
  }
  await nextTick()
  danmu_list.value.scrollTo({ top: danmu_list.value.scrollHeight, behavior: 'smooth' })
}

const sc_list = ref(null)
const sendSc = async (data) => {
  let msg_id
  if (!data) {
    msg_id = getUUID()
    data = {
      msg_id: msg_id,
      uname: '青云',
      uface: noFaceSrc,
      fans_medal_name: '巫女酱',
      fans_medal_level: 21,
      fans_medal_wearing_status: true,

      msg: '醒目留言内容' + msg_id,
      start_time: 1 * 1000,
      end_time: 5 * 1000,

      fade_out: false
    }
  } else {
    msg_id = data.msg_id
    data = Object.assign(data, {
      fade_out: false
    })
  }
  state.sc_list.push(data)

  setTimeout(async () => {
    for (let i = 0; i < state.sc_list.length; i++) {
      if (state.sc_list[i].msg_id === msg_id) {
        state.sc_list[i].fade_out = true

        setTimeout(() => {
          for (let i = 0; i < state.sc_list.length; i++) {
            if (state.sc_list[i].msg_id === msg_id) {
              state.sc_list.splice(i, 1)
            }
          }
        }, 1000)
        break
      }
    }
  }, data.end_time - data.start_time)

  await nextTick()
  sc_list.value.scrollTo({ top: sc_list.value.scrollHeight, behavior: 'smooth' })
}

const gift_list = ref(null)
const sendGift = async (data) => {
  if (!data) {
    data = {
      msg_id: getUUID(),

      uname: '青云',
      uface: noFaceSrc,
      fans_medal_name: '巫女酱',
      fans_medal_level: 21,

      gift_name: '礼物名',
      gift_num: 5,
      rmb: 5
    }
  }
  state.gift_list.push(data)
  if (state.gift_list.length >= 50) {
    state.gift_list.splice(0, 1)
  }
  await nextTick()
  gift_list.value.scrollTo({ top: gift_list.value.scrollHeight, behavior: 'smooth' })
}

let init_params = {
  code: '',
  timestamp: 0,
  room_id: 0,
  mid: 0,
  caller: 'bilibili',
  code_sign: ''
}

function connectWebSocketServer() {
  if (state.is_connect_websocket) {
    return
  }
  if (!init_params.code) {
    state.connect_message = '请提供身份码'
    return
  }

  let heartbeatInterval

  const socket = new WebSocket(serverUrl)
  socket.addEventListener('open', () => {
    console.log('[WebSocket]成功建立连接')

    state.is_connect_websocket = true
    state.connect_message = '正在连接至直播间'
    socket.send(
      JSON.stringify({
        type: 'init',
        data: init_params
      })
    )
  })

  const onClosed = () => {
    clearInterval(heartbeatInterval)
    state.is_connect_websocket = false
    state.is_connect_room = false
    state.connect_message = '连接失败，正在重连'
    console.error('[WebSocket]发生断连，5秒后尝试重连')
    setTimeout(() => {
      connectWebSocketServer()
    }, 5000)
  }
  socket.addEventListener('close', onClosed)
  // socket.addEventListener('error', onClosed)
  socket.addEventListener('message', (event) => {
    console.log('[WebSocket]收到消息：', event.data)
    const data = JSON.parse(event.data)
    switch (data.type) {
      case 'room': {
        if (data.code !== 0) {
          state.is_connect_room = false
          state.connect_message = '连接失败，正在重试，失败原因：' + data.msg
          console.error('[直播间]房间连接失败, 5秒后尝试重连，错误信息：', data.msg)
          setTimeout(() => {
            socket.send(
              JSON.stringify({
                type: 'init',
                data: init_params
              })
            )
          }, 5000)
          return
        }
        state.is_connect_room = true
        state.connect_message = '连接成功'
        console.log('[直播间]房间连接成功, 房间信息：', data.data)
        state.room_info = data.data

        clearInterval(heartbeatInterval)
        heartbeatInterval = setInterval(() => {
          socket.send(
            JSON.stringify({
              type: 'heartbeat'
            })
          )
        }, 5000)
        break
      }
      case 'danmu': {
        sendDanmu(data.data)
        break
      }
      case 'superchat': {
        sendSc(data.data)
        break
      }
      case 'gift': {
        sendGift(data.data)
        break
      }
      case 'guard': {
        sendMemberShip(data.data)
        break
      }
    }
  })
}

onMounted(() => {
  const query = new URLSearchParams(location.search)

  const timestamp = Number(query.get('Timestamp'))
  const room_id = Number(query.get('RoomId'))
  const mid = Number(query.get('Mid'))
  const caller = query.get('Caller')
  const code = query.get('Code')
  const code_sign = query.get('CodeSign')

  init_params = {
    timestamp,
    room_id,
    mid,
    caller,
    code,
    code_sign
  }
  console.log('身份码code：', code)
  console.log('身份信息:', init_params)

  connectWebSocketServer()
})
</script>

<template>
  <main>
    <!-- <div class="test-buttons">
      <button class="button" @click="sendDanmu">有人发弹幕</button>
      <button class="button" @click="sendSc">有人发SC</button>
      <button class="button" @click="sendGift">有人送礼</button>
      <button class="button" @click="sendMemberShip">有人上舰</button>
    </div> -->
    <div class="membership">
      <img
        class="membership-image"
        :src="state.display_membership.img_src"
        loop="false"
        :class="{
          hidden: !state.display_membership.playing
        }"
      />
      <div
        class="membership-user"
        :class="{
          hidden: !state.display_membership.playing,
          'fade-in': !state.display_membership.fade_out,
          'fade-out': state.display_membership.fade_out
        }"
      >
        <div class="membership-face-container">
          <img class="membership-face" :src="state.display_membership.uface" />
        </div>
        <div class="membership-name-container">
          <span class="membership-name">{{ state.display_membership.uname }}</span>
        </div>
      </div>
    </div>
    <div class="main-container">
      <div class="left-container">
        <div class="status-container">
          <div class="status-user" v-if="state.is_connect_room">
            <img class="status-face" :src="state.room_info.uface" />
            <div class="status-name">{{ state.room_info.uname }}</div>
          </div>
          <div class="status-msg">{{ state.connect_message }}</div>
        </div>
        <div class="danmu-list-container" ref="danmu_list">
          <div class="danmu-list">
            <div class="danmu-item" v-for="(item, index) in state.danmu_list" :key="item.msg_id">
              <div class="danmu-user">
                <img class="danmu-face" :src="item.uface" />
                <div class="danmu-name">{{ item.uname }}</div>
                <div class="danmu-medal" v-if="item.fans_medal_wearing_status">
                  <div class="danmu-medal-name">{{ item.fans_medal_name }}</div>
                  <div class="danmu-medal-level">{{ item.fans_medal_level }}</div>
                </div>
                <div>：</div>
              </div>
              <div class="danmu-msg">{{ item.msg }}</div>
              <div class="danmu-line" v-if="index != state.danmu_list.length - 1"></div>
            </div>
          </div>
        </div>
      </div>
      <div class="sc-list-container" ref="sc_list">
        <div class="sc-list">
          <div
            class="sc-item"
            v-for="item in state.sc_list"
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
      <div class="gift-list-container" ref="gift_list">
        <div class="gift-list">
          <div class="gift-item" v-for="item in state.gift_list" :key="item.msg_id">
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
    </div>
  </main>
</template>

<style scoped>
@keyframes fadeOut {
  0% {
    opacity: 1;
  }
  100% {
    opacity: 0;
  }
}

@keyframes fadeInFromLeft {
  0% {
    transform: translateX(-100%);
    opacity: 0;
  }
  100% {
    opacity: 1;
    transform: translateX(0);
  }
}

@keyframes fadeInFromBottom {
  0% {
    transform: translateY(100%);
    opacity: 0;
  }
  100% {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes fadeOutToRight {
  0% {
    transform: translateX(0);
    opacity: 1;
  }
  100% {
    opacity: 0;
    transform: translateX(100%);
  }
}

@keyframes fadeOutToBottom {
  0% {
    transform: translateY(0);
    opacity: 1;
  }
  100% {
    opacity: 0;
    transform: translateY(100%);
  }
}

@keyframes fadeInFromLeftPlus {
  0% {
    opacity: 0;
    transform: rotate(-45deg) translateX(-100%);
  }
  100% {
    opacity: 1;
    transform: rotate(0) translateX(0);
  }
}

.test-buttons {
  position: absolute;
  left: 40%;
  top: 50%;
  z-index: 1000;
}

.hidden {
  display: none !important;
}

.main-container {
  display: flex;
  justify-content: space-between;
  height: 100vh;
}

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

.left-container {
  width: 30%;
  height: 100%;
  margin: 0 3% 3% 3%;
  padding: 10px;
}

.status-container {
  width: 100%;
  margin: 10px 0;
}

.status-user {
  display: flex;
  align-items: center;
  font-size: 14px;
}

.status-face {
  width: 24px;
  height: 24px;
  border-radius: 24px;
  margin-right: 5px;
}

.status-name {
}

.status-msg {
}

.danmu-list-container {
  width: 100%;
  height: 80%;
  border: 2px solid rgb(110, 171, 211);
  overflow-y: hidden;
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

.danmu-msg {
  margin-top: 5px;
  font-size: 16px;
}

.danmu-line {
  width: 100%;
  height: 1px;
  background-color: rgb(110, 171, 211);
  margin-top: 5px;
}

.sc-list-container {
  width: 30%;
  margin: 3%;
  padding: 10px;
}

.sc-list {
}

.sc-list:last-child {
  margin-bottom: 0;
}

.sc-item {
  width: 100%;
  margin-bottom: 20px;
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

.gift-list-container {
  width: 30%;
  margin: 3%;
  padding: 10px;

  overflow-y: hidden;
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
