import { reactive, readonly } from 'vue'

export function createSubmitFeedback() {
  const state = reactive({ status: 'idle', action: '', message: '', code: '', registerId: '', registerNo: '' })

  function beginAttempt(action) {
    Object.assign(state, { status: 'submitting', action: action || '', message: '', code: '', registerId: '', registerNo: '' })
  }

  function succeed(result, message) {
    const registerId = typeof result?.registerId === 'string' ? result.registerId : ''
    const registerNo = typeof result?.registerNo === 'string' ? result.registerNo : ''
    const identifier = registerNo || registerId
    const defaultMessage = identifier
      ? ((state.action === 'register' ? '挂号成功' : '操作成功') + '，流水号 ' + identifier)
      : '操作成功'
    Object.assign(state, {
      status: 'success',
      message: typeof message === 'string' && message.trim() ? message.trim() : defaultMessage,
      code: '',
      registerId,
      registerNo,
    })
  }

  function fail(message, code = '') {
    Object.assign(state, {
      status: 'error',
      message: typeof message === 'string' && message.trim() ? message.trim() : '操作失败，请稍后重试',
      code: typeof code === 'string' ? code : '',
      registerId: '',
      registerNo: '',
    })
  }

  function reset() {
    Object.assign(state, { status: 'idle', action: '', message: '', code: '', registerId: '', registerNo: '' })
  }

  return { state: readonly(state), beginAttempt, succeed, fail, reset }
}
