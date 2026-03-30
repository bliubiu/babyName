'use client';

import { useToast } from './Toast';

interface ShareButtonsProps {
  title: string;
  text: string;
  url: string;
}

export default function ShareButtons({ title, text, url }: ShareButtonsProps) {
  const { showToast } = useToast();

  const shareToWeChat = () => {
    // 微信分享需要特殊处理，这里使用toast提示用户
    showToast('请复制链接到微信分享', 'info');
  };

  const shareToWeibo = () => {
    const weiboUrl = `http://service.weibo.com/share/share.php?url=${encodeURIComponent(url)}&title=${encodeURIComponent(title)}&content=${encodeURIComponent(text)}`;
    window.open(weiboUrl, '_blank', 'width=600,height=400');
  };

  const shareToQQ = () => {
    const qqUrl = `https://connect.qq.com/widget/shareqq/index.html?url=${encodeURIComponent(url)}&title=${encodeURIComponent(title)}&summary=${encodeURIComponent(text)}`;
    window.open(qqUrl, '_blank', 'width=600,height=400');
  };

  const shareToEmail = () => {
    const emailUrl = `mailto:?subject=${encodeURIComponent(title)}&body=${encodeURIComponent(text)}\n\n${encodeURIComponent(url)}`;
    window.location.href = emailUrl;
  };

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(url);
      showToast('链接已复制到剪贴板', 'success');
    } catch (err) {
      console.error('复制失败:', err);
      showToast('复制失败，请手动复制链接', 'error');
    }
  };

  return (
    <div className="flex flex-wrap gap-2 md:gap-3 mt-4 justify-center sm:justify-start">
      <button
        onClick={shareToWeChat}
        className="flex items-center gap-1 text-teal hover:text-crimson transition-all duration-300 px-3 py-1.5 rounded-md hover:bg-crimson/10 hover-lift hover-glow text-sm sm:text-base touch-manipulation active:scale-95"
        title="分享到微信"
      >
        💬 微信
      </button>
      <button
        onClick={shareToWeibo}
        className="flex items-center gap-1 text-teal hover:text-crimson transition-all duration-300 px-3 py-1.5 rounded-md hover:bg-crimson/10 hover-lift hover-glow text-sm sm:text-base touch-manipulation active:scale-95"
        title="分享到微博"
      >
        📱 微博
      </button>
      <button
        onClick={shareToQQ}
        className="flex items-center gap-1 text-teal hover:text-crimson transition-all duration-300 px-3 py-1.5 rounded-md hover:bg-crimson/10 hover-lift hover-glow text-sm sm:text-base touch-manipulation active:scale-95"
        title="分享到QQ"
      >
        🐧 QQ
      </button>
      <button
        onClick={shareToEmail}
        className="flex items-center gap-1 text-teal hover:text-crimson transition-all duration-300 px-3 py-1.5 rounded-md hover:bg-crimson/10 hover-lift hover-glow text-sm sm:text-base touch-manipulation active:scale-95"
        title="分享到邮件"
      >
        📧 邮件
      </button>
      <button
        onClick={copyToClipboard}
        className="flex items-center gap-1 text-teal hover:text-crimson transition-all duration-300 px-3 py-1.5 rounded-md hover:bg-crimson/10 hover-lift hover-glow text-sm sm:text-base touch-manipulation active:scale-95"
        title="复制链接"
      >
        📋 复制
      </button>
    </div>
  );
}
