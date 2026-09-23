// apps/artisan/src/lib/lessons.ts
//
// F15 digital literacy: static lesson content. The server (core-svc's
// domain.Lessons) owns only the codes, order, which lessons have a practice
// task and what evidence that task needs; everything the artisan reads is
// here, as i18n keys. A new lesson needs both places.

import type { MessageKey } from '@kalakriti/i18n';
import type { IconName } from '@kalakriti/icons';

export interface QuizOption {
  key: MessageKey;
  icon: IconName;
}

export interface LessonContent {
  code: string;
  icon: IconName;
  title: MessageKey;
  cards: readonly [MessageKey, MessageKey, MessageKey];
  /** Evidence the server checks: IMAGE/AUDIO upload, or 'CONFIRM' for a tap. */
  practice?: { key: MessageKey; evidence: 'IMAGE' | 'AUDIO' | 'CONFIRM' };
  quiz: { question: MessageKey; options: readonly QuizOption[]; correct: number };
}

export const LESSONS: readonly LessonContent[] = [
  {
    code: 'PHOTO',
    icon: 'camera',
    title: 'learn.PHOTO.title',
    cards: ['learn.PHOTO.c1', 'learn.PHOTO.c2', 'learn.PHOTO.c3'],
    practice: { key: 'learn.PHOTO.practice', evidence: 'IMAGE' },
    quiz: {
      question: 'learn.PHOTO.q',
      options: [
        { key: 'learn.PHOTO.a1', icon: 'eye-off' },
        { key: 'learn.PHOTO.a2', icon: 'image' },
        { key: 'learn.PHOTO.a3', icon: 'package' },
      ],
      correct: 1,
    },
  },
  {
    code: 'STORY',
    icon: 'microphone',
    title: 'learn.STORY.title',
    cards: ['learn.STORY.c1', 'learn.STORY.c2', 'learn.STORY.c3'],
    practice: { key: 'learn.STORY.practice', evidence: 'AUDIO' },
    quiz: {
      question: 'learn.STORY.q',
      options: [
        { key: 'learn.STORY.a1', icon: 'user' },
        { key: 'learn.STORY.a2', icon: 'dollar-sign' },
        { key: 'learn.STORY.a3', icon: 'phone' },
      ],
      correct: 0,
    },
  },
  {
    code: 'PRICE',
    icon: 'fair-price',
    title: 'learn.PRICE.title',
    cards: ['learn.PRICE.c1', 'learn.PRICE.c2', 'learn.PRICE.c3'],
    quiz: {
      question: 'learn.PRICE.q',
      options: [
        { key: 'learn.PRICE.a1', icon: 'trash' },
        { key: 'learn.PRICE.a2', icon: 'close' },
        { key: 'learn.PRICE.a3', icon: 'fair-price' },
      ],
      correct: 2,
    },
  },
  {
    code: 'ORDERS',
    icon: 'collective-order',
    title: 'learn.ORDERS.title',
    cards: ['learn.ORDERS.c1', 'learn.ORDERS.c2', 'learn.ORDERS.c3'],
    quiz: {
      question: 'learn.ORDERS.q',
      options: [
        { key: 'learn.ORDERS.a1', icon: 'check' },
        { key: 'learn.ORDERS.a2', icon: 'edit' },
        { key: 'learn.ORDERS.a3', icon: 'eye-off' },
      ],
      correct: 1,
    },
  },
  {
    code: 'PAYMENTS',
    icon: 'income-statement',
    title: 'learn.PAYMENTS.title',
    cards: ['learn.PAYMENTS.c1', 'learn.PAYMENTS.c2', 'learn.PAYMENTS.c3'],
    quiz: {
      question: 'learn.PAYMENTS.q',
      options: [
        { key: 'learn.PAYMENTS.a1', icon: 'close' },
        { key: 'learn.PAYMENTS.a2', icon: 'dollar-sign' },
        { key: 'learn.PAYMENTS.a3', icon: 'lock' },
      ],
      correct: 0,
    },
  },
  {
    code: 'SAFETY',
    icon: 'lock',
    title: 'learn.SAFETY.title',
    cards: ['learn.SAFETY.c1', 'learn.SAFETY.c2', 'learn.SAFETY.c3'],
    practice: { key: 'learn.SAFETY.practice', evidence: 'CONFIRM' },
    quiz: {
      question: 'learn.SAFETY.q',
      options: [
        { key: 'learn.SAFETY.a1', icon: 'message' },
        { key: 'learn.SAFETY.a2', icon: 'share' },
        { key: 'learn.SAFETY.a3', icon: 'lock' },
      ],
      correct: 2,
    },
  },
  {
    code: 'WHATSAPP',
    icon: 'whatsapp',
    title: 'learn.WHATSAPP.title',
    cards: ['learn.WHATSAPP.c1', 'learn.WHATSAPP.c2', 'learn.WHATSAPP.c3'],
    quiz: {
      question: 'learn.WHATSAPP.q',
      options: [
        { key: 'learn.WHATSAPP.a1', icon: 'link' },
        { key: 'learn.WHATSAPP.a2', icon: 'image' },
        { key: 'learn.WHATSAPP.a3', icon: 'bell' },
      ],
      correct: 0,
    },
  },
  {
    code: 'LOAN',
    icon: 'dollar-sign',
    title: 'learn.LOAN.title',
    cards: ['learn.LOAN.c1', 'learn.LOAN.c2', 'learn.LOAN.c3'],
    quiz: {
      question: 'learn.LOAN.q',
      options: [
        { key: 'learn.LOAN.a1', icon: 'warning' },
        { key: 'learn.LOAN.a2', icon: 'verified-artisan' },
        { key: 'learn.LOAN.a3', icon: 'users' },
      ],
      correct: 1,
    },
  },
];

export function lessonContent(code: string): LessonContent | undefined {
  return LESSONS.find((l) => l.code === code);
}
