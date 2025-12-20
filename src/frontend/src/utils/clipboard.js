/**
 * 复制文本到剪贴板的通用工具函数
 * 兼容 HTTP 和 HTTPS 环境
 */

/**
 * 复制文本到剪贴板
 * @param {string} text - 要复制的文本
 * @returns {Promise<boolean>} - 是否复制成功
 */
export const copyToClipboard = async (text) => {
  // 首先尝试现代 Clipboard API
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch (err) {
      console.warn('Clipboard API failed, falling back to execCommand:', err);
    }
  }
  
  // 备用方案：使用传统的 execCommand 方式
  const textArea = document.createElement('textarea');
  textArea.value = text;
  
  // 避免滚动到底部
  textArea.style.position = 'fixed';
  textArea.style.left = '-9999px';
  textArea.style.top = '-9999px';
  textArea.style.opacity = '0';
  
  document.body.appendChild(textArea);
  textArea.focus();
  textArea.select();
  
  try {
    const successful = document.execCommand('copy');
    return successful;
  } catch (err) {
    console.error('execCommand copy failed:', err);
    return false;
  } finally {
    document.body.removeChild(textArea);
  }
};

/**
 * 复制文本到剪贴板并显示消息提示
 * @param {string} text - 要复制的文本
 * @param {object} message - antd message 实例
 * @param {string} successMsg - 成功消息（可选）
 * @param {string} errorMsg - 失败消息（可选）
 * @returns {Promise<boolean>} - 是否复制成功
 */
export const copyWithMessage = async (text, message, successMsg = '已复制到剪贴板', errorMsg = '复制失败，请手动复制') => {
  const success = await copyToClipboard(text);
  if (success) {
    message.success(successMsg);
  } else {
    message.error(errorMsg);
  }
  return success;
};

export default copyToClipboard;
