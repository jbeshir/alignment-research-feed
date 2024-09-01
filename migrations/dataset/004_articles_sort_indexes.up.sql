ALTER TABLE `alignment_research_dataset`.`articles`
    ADD INDEX `date_published_desc_idx` (`date_published` DESC),
    ADD INDEX `source_idx` (`source`),
    ADD INDEX `source_desc_idx` (`source` DESC);
