'use client';

import React, { useState, useRef, useMemo } from 'react';

interface TagChipInputProps {
  /** Current list of tags */
  tags: string[];
  /** Called when tags change (add or remove) */
  onChange: (tags: string[]) => void;
  /** Previously-used tags for autocomplete suggestions */
  suggestions?: string[];
  /** Placeholder text when no tags exist */
  placeholder?: string;
  /** Maximum number of tags allowed */
  maxTags?: number;
  /** Maximum length of a single tag */
  maxTagLength?: number;
}

export default function TagChipInput({
  tags,
  onChange,
  suggestions = [],
  placeholder = 'Add a tag...',
  maxTags = 50,
  maxTagLength = 100,
}: TagChipInputProps) {
  const [inputValue, setInputValue] = useState('');
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const inputRef = useRef<HTMLInputElement>(null);

  // Filter suggestions based on current input, excluding already-selected tags
  const filteredSuggestions = useMemo(() => {
    if (inputValue.trim().length === 0) return [];
    const query = inputValue.trim().toLowerCase();
    return suggestions
      .filter((s) => s.toLowerCase().includes(query) && !tags.includes(s.toLowerCase()))
      .slice(0, 6);
  }, [inputValue, suggestions, tags]);

  const addTag = (rawTag: string) => {
    const tag = rawTag.trim().toLowerCase();
    if (tag.length === 0) return;
    if (tag.length > maxTagLength) return;
    if (tags.length >= maxTags) return;
    if (tags.includes(tag)) return; // deduplicate

    onChange([...tags, tag]);
    setInputValue('');
    setShowSuggestions(false);
    setHighlightedIndex(-1);
  };

  const removeTag = (tagToRemove: string) => {
    onChange(tags.filter((t) => t !== tagToRemove));
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault();
      if (highlightedIndex >= 0 && filteredSuggestions[highlightedIndex]) {
        addTag(filteredSuggestions[highlightedIndex]);
      } else {
        addTag(inputValue);
      }
    } else if (e.key === 'Backspace' && inputValue === '' && tags.length > 0) {
      // Remove last tag
      onChange(tags.slice(0, -1));
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      setHighlightedIndex((prev) => (prev < filteredSuggestions.length - 1 ? prev + 1 : 0));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setHighlightedIndex((prev) => (prev > 0 ? prev - 1 : filteredSuggestions.length - 1));
    } else if (e.key === 'Escape') {
      setShowSuggestions(false);
      setHighlightedIndex(-1);
    }
  };

  return (
    <div className="relative">
      {/* Chip container with inline input */}
      <div
        className="flex flex-wrap items-center gap-1.5 p-2 border rounded-lg
          bg-ic-input-bg border-ic-input-border
          focus-within:ring-2 focus-within:ring-ic-blue focus-within:border-ic-blue
          transition-colors min-h-[38px]"
        onClick={() => inputRef.current?.focus()}
      >
        {tags.map((tag) => (
          <span
            key={tag}
            className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full
              text-xs font-medium bg-ic-blue/10 text-ic-blue"
          >
            {tag}
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                removeTag(tag);
              }}
              className="hover:text-ic-blue-hover
                focus:outline-none focus:text-ic-blue-hover ml-0.5"
              aria-label={`Remove tag: ${tag}`}
            >
              &times;
            </button>
          </span>
        ))}
        <input
          ref={inputRef}
          type="text"
          value={inputValue}
          onChange={(e) => {
            setInputValue(e.target.value);
            setShowSuggestions(true);
            setHighlightedIndex(-1);
          }}
          onKeyDown={handleKeyDown}
          onFocus={() => setShowSuggestions(true)}
          onBlur={() => {
            // Delay to allow clicking suggestions
            setTimeout(() => setShowSuggestions(false), 150);
          }}
          placeholder={tags.length === 0 ? placeholder : ''}
          className="flex-1 min-w-[80px] outline-none bg-transparent text-sm
            text-ic-text-primary placeholder-ic-text-dim"
          aria-label="Add a tag. Type and press Enter to create."
          disabled={tags.length >= maxTags}
        />
      </div>

      {/* Tag suggestions dropdown */}
      {showSuggestions && filteredSuggestions.length > 0 && (
        <ul
          className="absolute z-50 mt-1 w-full bg-ic-bg-primary
            border border-ic-border rounded-lg shadow-lg shadow-black/20
            max-h-48 overflow-y-auto"
          role="listbox"
        >
          {filteredSuggestions.map((suggestion, index) => (
            <li
              key={suggestion}
              role="option"
              aria-selected={index === highlightedIndex}
              className={`px-3 py-2 text-sm cursor-pointer transition-colors
                ${
                  index === highlightedIndex
                    ? 'bg-ic-surface text-ic-blue'
                    : 'text-ic-text-secondary hover:bg-ic-surface/50'
                }`}
              onMouseDown={(e) => {
                e.preventDefault(); // Prevent blur
                addTag(suggestion);
              }}
              onMouseEnter={() => setHighlightedIndex(index)}
            >
              {suggestion}
            </li>
          ))}
        </ul>
      )}

      {/* Hint text */}
      {tags.length >= maxTags && (
        <p className="mt-1 text-xs text-ic-text-dim">Maximum {maxTags} tags reached.</p>
      )}
    </div>
  );
}
