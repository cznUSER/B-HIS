import { reactive, readonly } from 'vue'

const DIAGNOSIS_ACTIONS = new Set(['save', 'amend', 'commit', 'abate'])

export function createSubmitFeedback() {
  const state = reactive({
    status: 'idle',
    action: '',
    message: '',
    code: '',
    registerId: '',
    registerNo: '',
    diagnoseId: '',
  })

  function beginAttempt(action) {
    Object.assign(state, {
      status: 'submitting',
      action: action || '',
      message: '',
      code: '',
      registerId: '',
      registerNo: '',
      diagnoseId: '',
    })
  }

  function succeed(result, message) {
    const registerId = typeof result?.registerId === 'string' ? result.registerId : ''
    const registerNo = typeof result?.registerNo === 'string' ? result.registerNo : ''
    const diagnoseId = typeof result?.diagnoseId === 'string' ? result.diagnoseId : ''
    let defaultMessage = '操作成功'
    if (DIAGNOSIS_ACTIONS.has(state.action) && diagnoseId) {
      const statusHint = {
        save: '已暂存',
        amend: '修订已暂存',
        commit: '已提交',
        abate: '已作废',
      }[state.action] || '操作成功'
      defaultMessage = `诊断${statusHint}，诊断号 ${diagnoseId}`
    } else {
      const identifier = registerNo || registerId
      if (identifier) {
        defaultMessage = (state.action === 'register' ? '挂号成功' : '操作成功') + '，流水号 ' + identifier
      }
    }
    Object.assign(state, {
      status: 'success',
      message: typeof message === 'string' && message.trim() ? message.trim() : defaultMessage,
      code: '',
      registerId,
      registerNo,
      diagnoseId,
    })
  }

  function fail(message, code = '') {
    Object.assign(state, {
      status: 'error',
      message: typeof message === 'string' && message.trim() ? message.trim() : '操作失败，请稍后重试',
      code: typeof code === 'string' ? code : '',
      registerId: '',
      registerNo: '',
      diagnoseId: '',
    })
  }

  function reset() {
    Object.assign(state, {
      status: 'idle',
      action: '',
      message: '',
      code: '',
      registerId: '',
      registerNo: '',
      diagnoseId: '',
    })
  }

  return { state: readonly(state), beginAttempt, succeed, fail, reset }
}
