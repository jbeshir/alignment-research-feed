ALTER TABLE `alignment_research_dataset`.`articles`
    ADD FULLTEXT INDEX `title_fulltext_idx` (`title`);
