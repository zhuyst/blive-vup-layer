<script setup>
import { ref, watch, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import { useStore } from '@/store/live'

const store = useStore()
const { tts_list } = storeToRefs(store)

const audio_ref = ref(null)
const audio_src = ref('')

let audioPromise = null
let audioResolve = null
let isPlaying = false
const audioEnded = () => {
  if (!audioPromise) {
    return
  }
  audioResolve()
}

const real_tts_list = []
const playNextAudio = async () => {
  await audioPromise
  if (real_tts_list.length == 0) {
    isPlaying = false
    return
  }
  audio_src.value = real_tts_list.shift()
  audio_ref.value.load()
  audio_ref.value.play()
  audioPromise = new Promise((resolve) => {
    audioResolve = resolve
  })
  isPlaying = true
  playNextAudio()
}

watch(
  () => tts_list.value,
  async () => {
    await nextTick()
    if (tts_list.value.length == 0) {
      return
    }
    real_tts_list.push(tts_list.value[tts_list.value.length - 1].audio_file_path)
    if (isPlaying) {
      return
    }
    playNextAudio()
  },
  { deep: true }
)
</script>
<script>
export default {
  name: 'tts-audio'
}
</script>

<template>
  <div class="tts-audio">
    <audio controls id="audio" ref="audio_ref" @ended="audioEnded">
      <source type="audio/mpeg" id="audio_source" :src="audio_src" />
      <embed height="50" width="100" id="audio_embed" :src="audio_src" />
    </audio>
  </div>
</template>

<style scoped>
.tts-audio {
  position: absolute;
  right: 0;
  bottom: 0;
}
</style>
