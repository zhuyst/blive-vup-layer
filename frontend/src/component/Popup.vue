<script setup>
import { ref, onMounted } from 'vue';
import { defineEmits } from 'vue'

const inputValue = ref('')
const emit = defineEmits(['confirm', 'close'])

onMounted(() => {
  const savedCode = localStorage.getItem('savedCode');
  if (savedCode) {
    inputValue.value = savedCode;
  }
});

const handleConfirm = () => {
  localStorage.setItem('savedCode', inputValue.value);
  emit('confirm', inputValue.value);
};

const handleClose = () => {
  emit('close')
}
</script>

<template>
  <div class="modal">
    <div class="popup">
      <p>
        <span>身份码获取位置：</span>
        <a href="https://play-live.bilibili.com/" target="_blank"
          >https://play-live.bilibili.com/</a
        >
      </p>
      <div class="input-group">
        <label for="code">身份码：</label>
        <input v-model="inputValue" type="text" id="code" placeholder="请输入身份码" />
      </div>
      <div class="button-group">
        <button @click="handleConfirm">确定</button>
        <button class="close-button" @click="handleClose">关闭</button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'popup-component'
}
</script>

<style scoped>
.modal {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(5px);
  display: flex;
  justify-content: center;
  align-items: center;
}

.popup {
  background-color: white;
  padding: 25px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  width: 400px;
}

.input-group {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
}

.input-group label {
  margin-right: 10px;
  white-space: nowrap;
}

.input-group input {
  flex: 1;
  padding: 10px;
  border: 1px solid #ccc;
  border-radius: 4px;
}

.button-group {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.button-group button {
  padding: 10px 20px;
  background-color: #42b983;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.button-group button:hover {
  background-color: #369f6b;
}

.button-group .close-button {
  background-color: #ff4d4f;
}

.button-group .close-button:hover {
  background-color: #d9363e;
}
</style>
