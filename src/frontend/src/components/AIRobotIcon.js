import React from 'react';

// 美化的AI机器人图标组件 - 增强版特效
const AIRobotIcon = ({ size = 24, animated = true, className = '' }) => {
  return (
    <div className={`ai-robot-icon ${className}`} style={{ width: size, height: size }}>
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className={animated ? 'ai-robot-animated' : ''}
      >
        {/* 外发光效果 */}
        <defs>
          <filter id="glow">
            <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
            <feMerge>
              <feMergeNode in="coloredBlur"/>
              <feMergeNode in="SourceGraphic"/>
            </feMerge>
          </filter>

          <linearGradient id="robotGradient" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#e6f7ff">
              <animate attributeName="stop-color" values="#e6f7ff;#bae7ff;#e6f7ff" dur="3s" repeatCount="indefinite"/>
            </stop>
            <stop offset="100%" stopColor="#bae7ff">
              <animate attributeName="stop-color" values="#bae7ff;#91d5ff;#bae7ff" dur="3s" repeatCount="indefinite"/>
            </stop>
          </linearGradient>

          <linearGradient id="robotBodyGradient" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#f6ffed">
              <animate attributeName="stop-color" values="#f6ffed;#d9f7be;#f6ffed" dur="4s" repeatCount="indefinite"/>
            </stop>
            <stop offset="100%" stopColor="#d9f7be">
              <animate attributeName="stop-color" values="#d9f7be;#b7eb8f;#d9f7be" dur="4s" repeatCount="indefinite"/>
            </stop>
          </linearGradient>

          <radialGradient id="eyeGradient" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor="#fff"/>
            <stop offset="100%" stopColor="#1890ff"/>
          </radialGradient>
        </defs>

        {/* 机器人头部阴影 */}
        <ellipse cx="12" cy="19" rx="9" ry="2" fill="#000" opacity="0.1" />

        {/* 机器人头部 */}
        <rect
          x="4"
          y="6"
          width="16"
          height="12"
          rx="3"
          fill="url(#robotGradient)"
          stroke="#1890ff"
          strokeWidth="0.5"
          filter="url(#glow)"
        />

        {/* 眼睛 - 带闪烁动画 */}
        <circle cx="8" cy="10" r="1.5" fill="#fff" className="eye-white" />
        <circle cx="16" cy="10" r="1.5" fill="#fff" className="eye-white" />
        <circle cx="8" cy="10" r="0.8" fill="url(#eyeGradient)" className="eye-pupil" />
        <circle cx="16" cy="10" r="0.8" fill="url(#eyeGradient)" className="eye-pupil" />

        {/* 嘴巴 - 微笑动画 */}
        <path
          d="M10 14 Q12 16 14 14"
          stroke="#1890ff"
          strokeWidth="1.5"
          strokeLinecap="round"
          fill="none"
          className="mouth"
        />

        {/* 天线 - 摆动动画 */}
        <line x1="12" y1="6" x2="12" y2="4" stroke="#1890ff" strokeWidth="1.5" className="antenna-line" />
        <circle cx="12" cy="3" r="1" fill="#52c41a" className="antenna-tip">
          <animate attributeName="fill" values="#52c41a;#f5222d;#52c41a" dur="2s" repeatCount="indefinite"/>
        </circle>

        {/* 身体 */}
        <rect
          x="6"
          y="18"
          width="12"
          height="4"
          rx="2"
          fill="url(#robotBodyGradient)"
          stroke="#1890ff"
          strokeWidth="0.5"
        />

        {/* 手臂 - 轻微摆动 */}
        <rect x="2" y="19" width="4" height="2" rx="1" fill="#1890ff" className="arm-left" />
        <rect x="18" y="19" width="4" height="2" rx="1" fill="#1890ff" className="arm-right" />

        {/* 装饰性光点 */}
        <circle cx="6" cy="8" r="0.3" fill="#fff" opacity="0.8" className="decoration-dot dot1" />
        <circle cx="18" cy="8" r="0.3" fill="#fff" opacity="0.8" className="decoration-dot dot2" />
        <circle cx="6" cy="16" r="0.3" fill="#fff" opacity="0.8" className="decoration-dot dot3" />
        <circle cx="18" cy="16" r="0.3" fill="#fff" opacity="0.8" className="decoration-dot dot4" />
      </svg>

      <style jsx>{`
        .ai-robot-icon {
          display: inline-block;
          transition: all 0.3s ease;
          position: relative;
        }

        .ai-robot-icon:hover {
          transform: scale(1.1) rotate(2deg);
          filter: drop-shadow(0 4px 8px rgba(24, 144, 255, 0.3));
        }

        .ai-robot-animated {
          animation: robotFloat 3s ease-in-out infinite;
        }

        .ai-robot-animated:hover {
          animation: robotDance 0.8s ease-in-out infinite;
        }

        /* 漂浮动画 */
        @keyframes robotFloat {
          0%, 100% {
            transform: translateY(0px) scale(1);
          }
          50% {
            transform: translateY(-2px) scale(1.02);
          }
        }

        /* 跳舞动画 */
        @keyframes robotDance {
          0%, 100% {
            transform: scale(1.1) rotate(2deg);
          }
          25% {
            transform: scale(1.15) rotate(-2deg);
          }
          50% {
            transform: scale(1.1) rotate(2deg);
          }
          75% {
            transform: scale(1.15) rotate(-2deg);
          }
        }

        /* 眼睛闪烁动画 */
        .eye-pupil {
          animation: eyeBlink 4s ease-in-out infinite;
        }

        @keyframes eyeBlink {
          0%, 90%, 100% {
            opacity: 1;
            transform: scale(1);
          }
          95% {
            opacity: 0.3;
            transform: scale(0.8);
          }
        }

        /* 嘴巴微笑动画 */
        .mouth {
          animation: mouthSmile 5s ease-in-out infinite;
        }

        @keyframes mouthSmile {
          0%, 100% {
            d: path('M10 14 Q12 16 14 14');
          }
          50% {
            d: path('M10 14 Q12 17 14 14');
          }
        }

        /* 天线摆动动画 */
        .antenna-line {
          animation: antennaWave 3s ease-in-out infinite;
          transform-origin: 50% 100%;
        }

        .antenna-tip {
          animation: antennaGlow 2s ease-in-out infinite;
        }

        @keyframes antennaWave {
          0%, 100% {
            transform: rotate(0deg);
          }
          25% {
            transform: rotate(5deg);
          }
          75% {
            transform: rotate(-5deg);
          }
        }

        /* 手臂轻微摆动 */
        .arm-left {
          animation: armWaveLeft 4s ease-in-out infinite;
          transform-origin: 100% 50%;
        }

        .arm-right {
          animation: armWaveRight 4s ease-in-out infinite;
          transform-origin: 0% 50%;
        }

        @keyframes armWaveLeft {
          0%, 100% {
            transform: rotate(0deg);
          }
          50% {
            transform: rotate(-3deg);
          }
        }

        @keyframes armWaveRight {
          0%, 100% {
            transform: rotate(0deg);
          }
          50% {
            transform: rotate(3deg);
          }
        }

        /* 装饰点闪烁 */
        .decoration-dot {
          animation: dotTwinkle 3s ease-in-out infinite;
        }

        .dot1 { animation-delay: 0s; }
        .dot2 { animation-delay: 0.5s; }
        .dot3 { animation-delay: 1s; }
        .dot4 { animation-delay: 1.5s; }

        @keyframes dotTwinkle {
          0%, 100% {
            opacity: 0.8;
            transform: scale(1);
          }
          50% {
            opacity: 1;
            transform: scale(1.2);
          }
        }

        /* 响应式调整 */
        @media (prefers-reduced-motion: reduce) {
          .ai-robot-animated,
          .ai-robot-animated:hover,
          .eye-pupil,
          .mouth,
          .antenna-line,
          .arm-left,
          .arm-right,
          .decoration-dot {
            animation: none;
          }
        }
      `}</style>
    </div>
  );
};

export default AIRobotIcon;
