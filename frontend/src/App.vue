<script setup>
import { onMounted, reactive } from 'vue'
import { useStore } from '@/store/live'
import Membership from '@/component/Membership.vue'
import DanmuList from '@/component/DanmuList.vue'
import ScList from '@/component/ScList.vue'
import GiftList from '@/component/GiftList.vue'
import noFaceSrc from '@/assets/noface.gif'

let protocol = 'ws'
if (location.protocol === 'https:') {
  protocol = 'wss'
}

let serverUrl
if (import.meta.env.DEV) {
  serverUrl = protocol + '://localhost:8080/server/ws'
} else {
  serverUrl = protocol + '://' + location.host + '/server/ws'
}

const store = useStore()
const { sendMemberShip, sendDanmu, sendSc, sendGift } = store

const state = reactive({
  is_connect_websocket: false,
  is_connect_room: false,
  connect_message: '正在连接至直播间',

  room_info: {
    room_id: 0,
    uname: '',
    uface: noFaceSrc
  }
})

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
    <div class="test-buttons">
      <button class="button" @click="sendDanmu()">有人发弹幕</button>
      <button class="button" @click="sendSc()">有人发SC</button>
      <button class="button" @click="sendGift()">有人送礼</button>
      <button class="button" @click="sendMemberShip()">有人上舰</button>
    </div>
    <Membership />
    <div class="main-container">
      <div class="left-container">
        <div class="status-container">
          <div class="status-user" v-if="state.is_connect_room">
            <img class="status-face" :src="state.room_info.uface" />
            <div class="status-name">{{ state.room_info.uname }}</div>
          </div>
          <div class="status-msg">{{ state.connect_message }}</div>
        </div>
        <DanmuList />
      </div>
      <ScList />
      <GiftList />
    </div>
  </main>
</template>

<style scoped>
.test-buttons {
  position: absolute;
  left: 40%;
  top: 50%;
  z-index: 1000;
}

.main-container {
  display: flex;
  justify-content: space-between;
  height: 100vh;
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
  padding-top: 10px;
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
</style>
